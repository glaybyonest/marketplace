package postgres

import (
	"context"
	"encoding/json"

	"marketplace-backend/internal/observability"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ErrorEventRepository struct {
	db *pgxpool.Pool
}

func NewErrorEventRepository(db *pgxpool.Pool) *ErrorEventRepository {
	return &ErrorEventRepository{db: db}
}

func (r *ErrorEventRepository) Create(ctx context.Context, event observability.ErrorEvent) error {
	var userID any
	if event.UserID != nil {
		userID = *event.UserID
	}

	details := event.Details
	if details == nil {
		details = map[string]any{}
	}
	detailsRaw, err := json.Marshal(details)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO error_events (
			user_id,
			severity,
			code,
			message,
			request_id,
			method,
			path,
			route,
			status,
			details
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
	`, userID, event.Severity, event.Code, event.Message, event.RequestID, event.Method, event.Path, event.Route, event.Status, detailsRaw)
	return mapError(err)
}
