-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS cart_items (
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
	quantity INT NOT NULL CHECK (quantity > 0),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (user_id, product_id)
);
CREATE INDEX IF NOT EXISTS idx_cart_items_user_id ON cart_items(user_id);
CREATE INDEX IF NOT EXISTS idx_cart_items_product_id ON cart_items(product_id);

CREATE TABLE IF NOT EXISTS orders (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	place_id UUID NOT NULL,
	place_title TEXT NOT NULL,
	address_text TEXT NOT NULL,
	lat DOUBLE PRECISION NULL,
	lon DOUBLE PRECISION NULL,
	status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'cancelled')),
	currency CHAR(3) NOT NULL DEFAULT 'RUB',
	total_amount NUMERIC(12, 2) NOT NULL CHECK (total_amount >= 0),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);

CREATE TABLE IF NOT EXISTS order_items (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	product_id UUID NOT NULL,
	product_name TEXT NOT NULL,
	sku TEXT NOT NULL,
	unit_price NUMERIC(12, 2) NOT NULL CHECK (unit_price >= 0),
	quantity INT NOT NULL CHECK (quantity > 0),
	line_total NUMERIC(12, 2) NOT NULL CHECK (line_total >= 0),
	currency CHAR(3) NOT NULL DEFAULT 'RUB',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);

CREATE TRIGGER trg_cart_items_updated_at
	BEFORE UPDATE ON cart_items
	FOR EACH ROW
	EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_orders_updated_at
	BEFORE UPDATE ON orders
	FOR EACH ROW
	EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_orders_updated_at ON orders;
DROP TRIGGER IF EXISTS trg_cart_items_updated_at ON cart_items;

DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS cart_items;
-- +goose StatementEnd
