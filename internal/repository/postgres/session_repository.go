package postgres

import (
	"context"
	"fmt"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/usecase"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, input usecase.CreateSessionInput) (domain.UserSession, error) {
	const q = `
		INSERT INTO user_sessions (user_id, refresh_token_hash, user_agent, ip, expires_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, refresh_token_hash, COALESCE(user_agent, ''), COALESCE(ip::text, ''), created_at, last_seen_at, expires_at, revoked_at, rotated_at
	`

	var session domain.UserSession
	err := r.db.QueryRow(ctx, q, input.UserID, input.RefreshTokenHash, nullIfEmpty(input.UserAgent), nullIfEmpty(input.IP), input.ExpiresAt, time.Now().UTC()).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.UserAgent,
		&session.IP,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.RotatedAt,
	)
	if err != nil {
		return domain.UserSession{}, mapError(err)
	}
	return session, nil
}

func (r *SessionRepository) GetByRefreshTokenHash(ctx context.Context, tokenHash string) (domain.UserSession, error) {
	const q = `
		SELECT id, user_id, refresh_token_hash, COALESCE(user_agent, ''), COALESCE(ip::text, ''), created_at, last_seen_at, expires_at, revoked_at, rotated_at
		FROM user_sessions
		WHERE refresh_token_hash = $1
	`

	var session domain.UserSession
	err := r.db.QueryRow(ctx, q, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.UserAgent,
		&session.IP,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.RotatedAt,
	)
	if err != nil {
		return domain.UserSession{}, mapError(err)
	}
	return session, nil
}

func (r *SessionRepository) Rotate(
	ctx context.Context,
	oldSessionID uuid.UUID,
	oldTokenHash string,
	rotatedAt time.Time,
	newSession usecase.CreateSessionInput,
) (domain.UserSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.UserSession{}, fmt.Errorf("begin session rotation transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	cmd, err := tx.Exec(ctx, `
		UPDATE user_sessions
		SET rotated_at = $3
		WHERE id = $1 AND refresh_token_hash = $2 AND revoked_at IS NULL AND rotated_at IS NULL
	`, oldSessionID, oldTokenHash, rotatedAt)
	if err != nil {
		return domain.UserSession{}, mapError(err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.UserSession{}, domain.ErrTokenReused
	}

	var session domain.UserSession
	err = tx.QueryRow(ctx, `
		INSERT INTO user_sessions (user_id, refresh_token_hash, user_agent, ip, expires_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, refresh_token_hash, COALESCE(user_agent, ''), COALESCE(ip::text, ''), created_at, last_seen_at, expires_at, revoked_at, rotated_at
	`, newSession.UserID, newSession.RefreshTokenHash, nullIfEmpty(newSession.UserAgent), nullIfEmpty(newSession.IP), newSession.ExpiresAt, rotatedAt).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.UserAgent,
		&session.IP,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.RotatedAt,
	)
	if err != nil {
		return domain.UserSession{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.UserSession{}, fmt.Errorf("commit session rotation transaction: %w", err)
	}
	return session, nil
}

func (r *SessionRepository) RevokeByRefreshTokenHash(
	ctx context.Context,
	userID uuid.UUID,
	tokenHash string,
	revokedAt time.Time,
) (bool, error) {
	cmd, err := r.db.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at = $3
		WHERE user_id = $1 AND refresh_token_hash = $2 AND revoked_at IS NULL
	`, userID, tokenHash, revokedAt)
	if err != nil {
		return false, mapError(err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *SessionRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID, revokedAt)
	return mapError(err)
}

func (r *SessionRepository) ListActiveByUserID(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.UserSession, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, refresh_token_hash, COALESCE(user_agent, ''), COALESCE(ip::text, ''), created_at, last_seen_at, expires_at, revoked_at, rotated_at
		FROM user_sessions
		WHERE user_id = $1
		  AND revoked_at IS NULL
		  AND rotated_at IS NULL
		  AND expires_at > $2
		ORDER BY last_seen_at DESC, created_at DESC
	`, userID, now)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	sessions := make([]domain.UserSession, 0)
	for rows.Next() {
		var session domain.UserSession
		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.RefreshTokenHash,
			&session.UserAgent,
			&session.IP,
			&session.CreatedAt,
			&session.LastSeenAt,
			&session.ExpiresAt,
			&session.RevokedAt,
			&session.RotatedAt,
		); err != nil {
			return nil, mapError(err)
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return sessions, nil
}

func (r *SessionRepository) RevokeByID(ctx context.Context, userID, sessionID uuid.UUID, revokedAt time.Time) (bool, error) {
	cmd, err := r.db.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at = $3
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, sessionID, userID, revokedAt)
	if err != nil {
		return false, mapError(err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *SessionRepository) TouchLastSeen(ctx context.Context, sessionID uuid.UUID, seenAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE user_sessions
		SET last_seen_at = GREATEST(last_seen_at, $2)
		WHERE id = $1
	`, sessionID, seenAt)
	return mapError(err)
}

func (r *SessionRepository) IsActive(ctx context.Context, sessionID uuid.UUID, now time.Time) (bool, error) {
	var active bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_sessions
			WHERE id = $1
			  AND revoked_at IS NULL
			  AND rotated_at IS NULL
			  AND expires_at > $2
		)
	`, sessionID, now).Scan(&active)
	if err != nil {
		return false, mapError(err)
	}
	return active, nil
}

func (r *SessionRepository) CleanupExpiredAndRevoked(ctx context.Context, now time.Time) (int64, error) {
	cmd, err := r.db.Exec(ctx, `
		DELETE FROM user_sessions
		WHERE expires_at < $1
		   OR (revoked_at IS NOT NULL AND revoked_at < $1 - INTERVAL '24 hours')
		   OR (rotated_at IS NOT NULL AND rotated_at < $1 - INTERVAL '24 hours')
	`, now)
	if err != nil {
		return 0, mapError(err)
	}
	return cmd.RowsAffected(), nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
