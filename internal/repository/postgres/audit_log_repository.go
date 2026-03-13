package postgres

import (
	"context"
	"encoding/json"

	"marketplace-backend/internal/observability"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLogRepository struct {
	db *pgxpool.Pool
}

func NewAuditLogRepository(db *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, entry observability.AuditEntry) error {
	meta := observability.RequestMetaFromContext(ctx)

	var actorUserID any
	if entry.ActorUserID != nil {
		actorUserID = *entry.ActorUserID
	}

	var entityID any
	if entry.EntityID != nil {
		entityID = *entry.EntityID
	}

	metadata := entry.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO audit_logs (
			actor_user_id,
			action,
			entity_type,
			entity_id,
			request_id,
			method,
			path,
			remote_ip,
			user_agent,
			metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
	`, actorUserID, entry.Action, entry.EntityType, entityID, meta.RequestID, meta.Method, meta.Path, meta.RemoteIP, meta.UserAgent, metadataRaw)
	return mapError(err)
}
