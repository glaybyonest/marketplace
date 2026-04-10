package postgres

import (
	"context"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/usecase"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversationReadStateRepository struct {
	db *pgxpool.Pool
}

func NewConversationReadStateRepository(db *pgxpool.Pool) *ConversationReadStateRepository {
	return &ConversationReadStateRepository{db: db}
}

func (r *ConversationReadStateRepository) Mark(ctx context.Context, input usecase.ConversationReadWriteInput) (domain.ConversationReadState, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO conversation_reads (
			conversation_id,
			user_id,
			last_read_message_id,
			last_read_at
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (conversation_id, user_id)
		DO UPDATE SET
			last_read_message_id = EXCLUDED.last_read_message_id,
			last_read_at = EXCLUDED.last_read_at
		RETURNING
			conversation_id,
			user_id,
			last_read_message_id,
			last_read_at
	`, input.ConversationID, input.UserID, input.LastReadMessageID, input.LastReadAt)

	var state domain.ConversationReadState
	if err := row.Scan(
		&state.ConversationID,
		&state.UserID,
		&state.LastReadMessageID,
		&state.LastReadAt,
	); err != nil {
		return domain.ConversationReadState{}, mapError(err)
	}
	return state, nil
}
