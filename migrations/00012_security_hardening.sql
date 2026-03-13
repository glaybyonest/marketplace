-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS failed_login_attempts INT NOT NULL DEFAULT 0,
	ADD COLUMN IF NOT EXISTS last_failed_login_at TIMESTAMPTZ NULL,
	ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ NULL;

ALTER TABLE users
	DROP CONSTRAINT IF EXISTS users_failed_login_attempts_nonnegative;

ALTER TABLE users
	ADD CONSTRAINT users_failed_login_attempts_nonnegative
	CHECK (failed_login_attempts >= 0);

CREATE INDEX IF NOT EXISTS idx_users_locked_until
	ON users (locked_until)
	WHERE locked_until IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_locked_until;

ALTER TABLE users
	DROP CONSTRAINT IF EXISTS users_failed_login_attempts_nonnegative;

ALTER TABLE users
	DROP COLUMN IF EXISTS locked_until,
	DROP COLUMN IF EXISTS last_failed_login_at,
	DROP COLUMN IF EXISTS failed_login_attempts;
-- +goose StatementEnd
