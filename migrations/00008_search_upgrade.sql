-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS search_queries (
	query_text TEXT PRIMARY KEY,
	search_count INT NOT NULL DEFAULT 1 CHECK (search_count > 0),
	last_searched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT search_queries_query_text_not_empty CHECK (BTRIM(query_text) <> '')
);

CREATE INDEX IF NOT EXISTS idx_search_queries_popular
	ON search_queries (search_count DESC, last_searched_at DESC);

CREATE INDEX IF NOT EXISTS idx_products_price_active
	ON products (price)
	WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_products_stock_active
	ON products (stock_qty)
	WHERE is_active = TRUE;

DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN
		EXECUTE 'CREATE INDEX IF NOT EXISTS idx_search_queries_query_trgm ON search_queries USING GIN (LOWER(query_text) gin_trgm_ops)';
	ELSE
		RAISE NOTICE 'Skipping idx_search_queries_query_trgm because pg_trgm is unavailable';
	END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_search_queries_query_trgm;
DROP INDEX IF EXISTS idx_products_stock_active;
DROP INDEX IF EXISTS idx_products_price_active;
DROP INDEX IF EXISTS idx_search_queries_popular;
DROP TABLE IF EXISTS search_queries;
-- +goose StatementEnd
