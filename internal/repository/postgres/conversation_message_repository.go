package postgres

import (
	"context"
	"slices"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/usecase"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversationMessageRepository struct {
	db *pgxpool.Pool
}

func NewConversationMessageRepository(db *pgxpool.Pool) *ConversationMessageRepository {
	return &ConversationMessageRepository{db: db}
}

func (r *ConversationMessageRepository) List(ctx context.Context, conversationID uuid.UUID, filter domain.ConversationMessageFilter) (domain.PageResult[domain.ConversationMessage], error) {
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM conversation_messages
		WHERE conversation_id = $1
	`, conversationID).Scan(&total); err != nil {
		return domain.PageResult[domain.ConversationMessage]{}, mapError(err)
	}

	offset := (filter.Page - 1) * filter.Limit
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			conversation_id,
			sender_id,
			body,
			created_at,
			edited_at
		FROM conversation_messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, conversationID, filter.Limit, offset)
	if err != nil {
		return domain.PageResult[domain.ConversationMessage]{}, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.ConversationMessage, 0, filter.Limit)
	for rows.Next() {
		var message domain.ConversationMessage
		if err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.Body,
			&message.CreatedAt,
			&message.EditedAt,
		); err != nil {
			return domain.PageResult[domain.ConversationMessage]{}, mapError(err)
		}
		items = append(items, message)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.ConversationMessage]{}, mapError(err)
	}

	slices.Reverse(items)

	return domain.PageResult[domain.ConversationMessage]{
		Items: items,
		Page:  filter.Page,
		Limit: filter.Limit,
		Total: total,
	}, nil
}

func (r *ConversationMessageRepository) Send(ctx context.Context, input usecase.ConversationMessageWriteInput) (domain.ConversationMessage, error) {
	row := r.db.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO conversation_messages (
				conversation_id,
				sender_id,
				body
			)
			VALUES ($1, $2, $3)
			RETURNING
				id,
				conversation_id,
				sender_id,
				body,
				created_at,
				edited_at
		),
		updated_conversation AS (
			UPDATE conversations c
			SET
				last_message_at = inserted.created_at,
				last_message_preview = $4,
				updated_at = NOW()
			FROM inserted
			WHERE c.id = inserted.conversation_id
		),
		updated_read_state AS (
			INSERT INTO conversation_reads (
				conversation_id,
				user_id,
				last_read_message_id,
				last_read_at
			)
			SELECT
				conversation_id,
				sender_id,
				id,
				created_at
			FROM inserted
			ON CONFLICT (conversation_id, user_id)
			DO UPDATE SET
				last_read_message_id = EXCLUDED.last_read_message_id,
				last_read_at = EXCLUDED.last_read_at
		)
		SELECT
			id,
			conversation_id,
			sender_id,
			body,
			created_at,
			edited_at
		FROM inserted
	`, input.ConversationID, input.SenderID, input.Body, input.Preview)

	var message domain.ConversationMessage
	if err := row.Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Body,
		&message.CreatedAt,
		&message.EditedAt,
	); err != nil {
		return domain.ConversationMessage{}, mapError(err)
	}
	return message, nil
}

func (r *ConversationMessageRepository) GetLatest(ctx context.Context, conversationID uuid.UUID) (domain.ConversationMessage, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			id,
			conversation_id,
			sender_id,
			body,
			created_at,
			edited_at
		FROM conversation_messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, conversationID)

	var message domain.ConversationMessage
	if err := row.Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Body,
		&message.CreatedAt,
		&message.EditedAt,
	); err != nil {
		return domain.ConversationMessage{}, mapError(err)
	}
	return message, nil
}
