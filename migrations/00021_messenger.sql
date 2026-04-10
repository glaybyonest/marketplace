-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS conversations (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	product_id UUID NOT NULL REFERENCES products(id),
	seller_id UUID NOT NULL REFERENCES users(id),
	buyer_id UUID NOT NULL REFERENCES users(id),
	order_id UUID NULL REFERENCES orders(id),
	last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	last_message_preview TEXT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT conversations_buyer_seller_product_unique UNIQUE (product_id, seller_id, buyer_id),
	CONSTRAINT conversations_buyer_not_seller CHECK (buyer_id <> seller_id)
);

CREATE TABLE IF NOT EXISTS conversation_messages (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
	sender_id UUID NOT NULL REFERENCES users(id),
	body TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	edited_at TIMESTAMPTZ NULL,
	CONSTRAINT conversation_messages_body_not_empty CHECK (BTRIM(body) <> '')
);

CREATE TABLE IF NOT EXISTS conversation_reads (
	conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
	user_id UUID NOT NULL REFERENCES users(id),
	last_read_message_id UUID NULL REFERENCES conversation_messages(id),
	last_read_at TIMESTAMPTZ NULL,
	PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_conversations_seller_last_message_at
	ON conversations (seller_id, last_message_at DESC);

CREATE INDEX IF NOT EXISTS idx_conversations_buyer_last_message_at
	ON conversations (buyer_id, last_message_at DESC);

CREATE INDEX IF NOT EXISTS idx_conversation_messages_conversation_created_at
	ON conversation_messages (conversation_id, created_at ASC);

CREATE TRIGGER trg_conversations_updated_at
	BEFORE UPDATE ON conversations
	FOR EACH ROW
	EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_conversations_updated_at ON conversations;

DROP INDEX IF EXISTS idx_conversation_messages_conversation_created_at;
DROP INDEX IF EXISTS idx_conversations_buyer_last_message_at;
DROP INDEX IF EXISTS idx_conversations_seller_last_message_at;

DROP TABLE IF EXISTS conversation_reads;
DROP TABLE IF EXISTS conversation_messages;
DROP TABLE IF EXISTS conversations;
-- +goose StatementEnd
