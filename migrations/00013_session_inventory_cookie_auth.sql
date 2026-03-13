-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_sessions
	ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE user_sessions
SET last_seen_at = created_at
WHERE last_seen_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_last_seen
	ON user_sessions (user_id, last_seen_at DESC, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_user_sessions_user_last_seen;

ALTER TABLE user_sessions
	DROP COLUMN IF EXISTS last_seen_at;
-- +goose StatementEnd
