-- +goose Up
-- +goose StatementBegin
UPDATE products
SET
	gallery = CASE
		WHEN COALESCE(BTRIM(image_url), '') = '' THEN
			CASE
				WHEN jsonb_typeof(gallery) = 'array' THEN gallery
				ELSE '[]'::jsonb
			END
		WHEN jsonb_typeof(gallery) = 'array'
			AND gallery @> jsonb_build_array(BTRIM(image_url)) THEN gallery
		WHEN jsonb_typeof(gallery) = 'array' THEN jsonb_build_array(BTRIM(image_url)) || gallery
		ELSE jsonb_build_array(BTRIM(image_url))
	END,
	specs = CASE
		WHEN jsonb_typeof(specs) = 'object' THEN specs
		ELSE '{}'::jsonb
	END;

ALTER TABLE products
	DROP CONSTRAINT IF EXISTS products_gallery_is_array;

ALTER TABLE products
	ADD CONSTRAINT products_gallery_is_array
	CHECK (jsonb_typeof(gallery) = 'array')
	NOT VALID;
ALTER TABLE products VALIDATE CONSTRAINT products_gallery_is_array;

ALTER TABLE products
	DROP CONSTRAINT IF EXISTS products_specs_is_object;

ALTER TABLE products
	ADD CONSTRAINT products_specs_is_object
	CHECK (jsonb_typeof(specs) = 'object')
	NOT VALID;
ALTER TABLE products VALIDATE CONSTRAINT products_specs_is_object;

ALTER TABLE products
	DROP COLUMN image_url;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE products
	ADD COLUMN IF NOT EXISTS image_url TEXT NULL;

UPDATE products
SET image_url = CASE
	WHEN jsonb_typeof(gallery) = 'array' THEN NULLIF(BTRIM(gallery->>0), '')
	ELSE NULL
END;

ALTER TABLE products
	DROP CONSTRAINT IF EXISTS products_specs_is_object;

ALTER TABLE products
	DROP CONSTRAINT IF EXISTS products_gallery_is_array;
-- +goose StatementEnd
