-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS audit_logs (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	actor_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
	action TEXT NOT NULL,
	entity_type TEXT NOT NULL,
	entity_id UUID NULL,
	request_id TEXT NULL,
	method TEXT NULL,
	path TEXT NULL,
	remote_ip TEXT NULL,
	user_agent TEXT NULL,
	metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_user_id ON audit_logs(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);

CREATE TABLE IF NOT EXISTS error_events (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
	severity TEXT NOT NULL CHECK (severity IN ('error', 'panic')),
	code TEXT NOT NULL,
	message TEXT NOT NULL,
	request_id TEXT NULL,
	method TEXT NULL,
	path TEXT NULL,
	route TEXT NULL,
	status INT NULL,
	details JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_error_events_user_id ON error_events(user_id);
CREATE INDEX IF NOT EXISTS idx_error_events_severity ON error_events(severity);
CREATE INDEX IF NOT EXISTS idx_error_events_created_at ON error_events(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS error_events;
DROP TABLE IF EXISTS audit_logs;
-- +goose StatementEnd
