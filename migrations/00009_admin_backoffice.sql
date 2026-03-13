-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
	ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'customer';

UPDATE users
SET role = 'customer'
WHERE BTRIM(role) = '';

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conname = 'users_role_allowed'
	) THEN
		ALTER TABLE users
			ADD CONSTRAINT users_role_allowed CHECK (role IN ('customer', 'admin'));
	END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
	DROP CONSTRAINT IF EXISTS users_role_allowed,
	DROP COLUMN IF EXISTS role;
-- +goose StatementEnd
