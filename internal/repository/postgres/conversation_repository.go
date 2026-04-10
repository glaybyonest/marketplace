package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/usecase"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversationRepository struct {
	db *pgxpool.Pool
}

func NewConversationRepository(db *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) Start(ctx context.Context, input usecase.ConversationStartWriteInput) (domain.Conversation, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Conversation{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var conversationID uuid.UUID
	insertErr := tx.QueryRow(ctx, `
		INSERT INTO conversations (
			product_id,
			seller_id,
			buyer_id,
			order_id
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (product_id, seller_id, buyer_id)
		DO NOTHING
		RETURNING id
	`, input.ProductID, input.SellerID, input.BuyerID, input.OrderID).Scan(&conversationID)

	createdConversation := insertErr == nil
	if insertErr != nil {
		if !errors.Is(insertErr, pgx.ErrNoRows) {
			return domain.Conversation{}, mapError(insertErr)
		}
		if err := tx.QueryRow(ctx, `
			SELECT id
			FROM conversations
			WHERE product_id = $1
			  AND seller_id = $2
			  AND buyer_id = $3
		`, input.ProductID, input.SellerID, input.BuyerID).Scan(&conversationID); err != nil {
			return domain.Conversation{}, mapError(err)
		}
	}

	if createdConversation {
		buyerReadAt := time.Now().UTC()
		var initialMessageID *uuid.UUID

		if input.InitialBody != "" {
			var messageID uuid.UUID
			var messageCreatedAt time.Time
			if err := tx.QueryRow(ctx, `
				INSERT INTO conversation_messages (
					conversation_id,
					sender_id,
					body
				)
				VALUES ($1, $2, $3)
				RETURNING id, created_at
			`, conversationID, input.BuyerID, input.InitialBody).Scan(&messageID, &messageCreatedAt); err != nil {
				return domain.Conversation{}, mapError(err)
			}

			initialMessageID = &messageID
			buyerReadAt = messageCreatedAt

			if _, err := tx.Exec(ctx, `
				UPDATE conversations
				SET
					last_message_at = $2,
					last_message_preview = $3,
					updated_at = NOW()
				WHERE id = $1
			`, conversationID, messageCreatedAt, truncateText(input.InitialBody, 160)); err != nil {
				return domain.Conversation{}, mapError(err)
			}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO conversation_reads (
				conversation_id,
				user_id,
				last_read_message_id,
				last_read_at
			)
			VALUES
				($1, $2, $3, $4),
				($1, $5, NULL, NULL)
			ON CONFLICT (conversation_id, user_id)
			DO NOTHING
		`, conversationID, input.BuyerID, initialMessageID, buyerReadAt, input.SellerID); err != nil {
			return domain.Conversation{}, mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Conversation{}, err
	}

	return r.GetByID(ctx, conversationID, input.BuyerID)
}

func (r *ConversationRepository) ListForUser(ctx context.Context, userID uuid.UUID, filter domain.ConversationFilter) (domain.PageResult[domain.Conversation], error) {
	condition := conversationRoleCondition(filter.Role, 1)

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM conversations c
		WHERE `+condition, userID).Scan(&total); err != nil {
		return domain.PageResult[domain.Conversation]{}, mapError(err)
	}

	offset := (filter.Page - 1) * filter.Limit
	rows, err := r.db.Query(ctx, `
		SELECT `+conversationSelectColumns(1)+`
		FROM conversations c
		`+conversationSelectJoins(1)+`
		WHERE `+condition+`
		ORDER BY c.last_message_at DESC, c.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, filter.Limit, offset)
	if err != nil {
		return domain.PageResult[domain.Conversation]{}, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.Conversation, 0, filter.Limit)
	for rows.Next() {
		var conversation domain.Conversation
		if err := scanConversation(rows, &conversation); err != nil {
			return domain.PageResult[domain.Conversation]{}, mapError(err)
		}
		items = append(items, conversation)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.Conversation]{}, mapError(err)
	}

	return domain.PageResult[domain.Conversation]{
		Items: items,
		Page:  filter.Page,
		Limit: filter.Limit,
		Total: total,
	}, nil
}

func (r *ConversationRepository) GetByID(ctx context.Context, id, currentUserID uuid.UUID) (domain.Conversation, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+conversationSelectColumns(1)+`
		FROM conversations c
		`+conversationSelectJoins(1)+`
		WHERE c.id = $2
	`, currentUserID, id)

	var conversation domain.Conversation
	if err := scanConversation(row, &conversation); err != nil {
		return domain.Conversation{}, mapError(err)
	}
	return conversation, nil
}

func (r *ConversationRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID, role domain.ConversationRole) (int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM conversation_messages m
		INNER JOIN conversations c ON c.id = m.conversation_id
		LEFT JOIN conversation_reads cr
			ON cr.conversation_id = c.id
		   AND cr.user_id = $1
		WHERE `+conversationRoleCondition(role, 1)+`
		  AND m.sender_id <> $1
		  AND (cr.last_read_at IS NULL OR m.created_at > cr.last_read_at)
	`, userID).Scan(&total); err != nil {
		return 0, mapError(err)
	}
	return total, nil
}

func conversationSelectColumns(currentUserPos int) string {
	return fmt.Sprintf(`
		c.id,
		c.product_id,
		p.name,
		COALESCE(NULLIF(p.gallery->>0, ''), ''),
		c.seller_id,
		COALESCE(NULLIF(seller_user.full_name, ''), seller_user.email, ''),
		COALESCE(seller_profile.store_name, ''),
		c.buyer_id,
		COALESCE(NULLIF(buyer_user.full_name, ''), buyer_user.email, ''),
		COALESCE(c.order_id::text, ''),
		c.last_message_at,
		COALESCE(c.last_message_preview, ''),
		COALESCE(unread_stats.unread_count, 0),
		CASE
			WHEN $%[1]d = c.seller_id THEN 'seller'
			WHEN $%[1]d = c.buyer_id THEN 'buyer'
			ELSE ''
		END,
		c.created_at,
		c.updated_at
	`, currentUserPos)
}

func conversationSelectJoins(currentUserPos int) string {
	return fmt.Sprintf(`
		INNER JOIN products p ON p.id = c.product_id
		INNER JOIN users seller_user ON seller_user.id = c.seller_id
		INNER JOIN users buyer_user ON buyer_user.id = c.buyer_id
		LEFT JOIN seller_profiles seller_profile ON seller_profile.user_id = c.seller_id
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::int AS unread_count
			FROM conversation_messages m
			LEFT JOIN conversation_reads cr
				ON cr.conversation_id = c.id
			   AND cr.user_id = $%[1]d
			WHERE m.conversation_id = c.id
			  AND m.sender_id <> $%[1]d
			  AND (cr.last_read_at IS NULL OR m.created_at > cr.last_read_at)
		) unread_stats ON TRUE
	`, currentUserPos)
}

func conversationRoleCondition(role domain.ConversationRole, userPos int) string {
	switch role {
	case domain.ConversationRoleBuyer:
		return fmt.Sprintf("c.buyer_id = $%d", userPos)
	case domain.ConversationRoleSeller:
		return fmt.Sprintf("c.seller_id = $%d", userPos)
	default:
		return fmt.Sprintf("(c.buyer_id = $%[1]d OR c.seller_id = $%[1]d)", userPos)
	}
}

func scanConversation(row pgx.Row, conversation *domain.Conversation) error {
	var orderIDText string
	var role string

	if err := row.Scan(
		&conversation.ID,
		&conversation.ProductID,
		&conversation.ProductName,
		&conversation.ProductImageURL,
		&conversation.SellerID,
		&conversation.SellerName,
		&conversation.SellerStoreName,
		&conversation.BuyerID,
		&conversation.BuyerName,
		&orderIDText,
		&conversation.LastMessageAt,
		&conversation.LastMessagePreview,
		&conversation.UnreadCount,
		&role,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	); err != nil {
		return err
	}

	if orderIDText != "" {
		orderID, err := uuid.Parse(orderIDText)
		if err != nil {
			return fmt.Errorf("parse order id: %w", err)
		}
		conversation.OrderID = &orderID
	}
	conversation.CurrentUserRole = domain.ConversationRole(role)
	return nil
}
