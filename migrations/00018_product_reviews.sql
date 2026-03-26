-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS product_reviews (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
	user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
	author_name TEXT NOT NULL,
	rating INT NOT NULL,
	comment TEXT NOT NULL,
	is_published BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT product_reviews_rating_range CHECK (rating BETWEEN 1 AND 5),
	CONSTRAINT product_reviews_author_name_not_empty CHECK (BTRIM(author_name) <> ''),
	CONSTRAINT product_reviews_comment_not_empty CHECK (BTRIM(comment) <> '')
);

CREATE INDEX IF NOT EXISTS idx_product_reviews_product_created_at
	ON product_reviews (product_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_reviews_user_product_unique
	ON product_reviews (user_id, product_id)
	WHERE user_id IS NOT NULL;

CREATE TRIGGER trg_product_reviews_updated_at
	BEFORE UPDATE ON product_reviews
	FOR EACH ROW
	EXECUTE FUNCTION set_updated_at();

INSERT INTO product_reviews (
	id,
	product_id,
	user_id,
	author_name,
	rating,
	comment,
	is_published,
	created_at
)
VALUES
	(
		'44444444-4444-4444-4444-444444444201',
		'22222222-2222-2222-2222-222222222201',
		NULL,
		'Olga',
		5,
		'Comfortable fit, clear sound and the noise reduction really helps on the metro.',
		TRUE,
		NOW() - INTERVAL '15 days'
	),
	(
		'44444444-4444-4444-4444-444444444202',
		'22222222-2222-2222-2222-222222222202',
		NULL,
		'Maxim',
		4,
		'Compact speaker with a surprisingly full sound. Battery lasts through a weekend trip.',
		TRUE,
		NOW() - INTERVAL '12 days'
	),
	(
		'44444444-4444-4444-4444-444444444203',
		'22222222-2222-2222-2222-222222222203',
		NULL,
		'Elena',
		5,
		'The charging pad is handy and the warm light mode is perfect for evening work.',
		TRUE,
		NOW() - INTERVAL '10 days'
	),
	(
		'44444444-4444-4444-4444-444444444204',
		'22222222-2222-2222-2222-222222222204',
		NULL,
		'Irina',
		5,
		'Stable mixer, easy to clean and handles dough without wobbling on the counter.',
		TRUE,
		NOW() - INTERVAL '9 days'
	),
	(
		'44444444-4444-4444-4444-444444444205',
		'22222222-2222-2222-2222-222222222205',
		NULL,
		'Pavel',
		4,
		'Grinds evenly and the settings are easy to switch. Noticeably quieter than my old one.',
		TRUE,
		NOW() - INTERVAL '8 days'
	),
	(
		'44444444-4444-4444-4444-444444444206',
		'22222222-2222-2222-2222-222222222206',
		NULL,
		'Natalia',
		5,
		'Good seal, stack neatly in the fridge and the glass does not keep food odors.',
		TRUE,
		NOW() - INTERVAL '7 days'
	),
	(
		'44444444-4444-4444-4444-444444444207',
		'22222222-2222-2222-2222-222222222207',
		NULL,
		'Roman',
		4,
		'Useful set for desk drawers and bathroom shelves. Modules combine nicely.',
		TRUE,
		NOW() - INTERVAL '6 days'
	),
	(
		'44444444-4444-4444-4444-444444444208',
		'22222222-2222-2222-2222-222222222208',
		NULL,
		'Anna',
		5,
		'Lightweight bag with enough pockets for daily essentials. Strap feels sturdy.',
		TRUE,
		NOW() - INTERVAL '5 days'
	),
	(
		'44444444-4444-4444-4444-444444444209',
		'22222222-2222-2222-2222-222222222209',
		NULL,
		'Dmitry',
		5,
		'Perfect cabin size and the shoe section keeps the rest of the bag organized.',
		TRUE,
		NOW() - INTERVAL '4 days'
	),
	(
		'44444444-4444-4444-4444-444444444210',
		'22222222-2222-2222-2222-222222222210',
		NULL,
		'Svetlana',
		4,
		'Comfortable cushioning and it does not slip during home workouts.',
		TRUE,
		NOW() - INTERVAL '3 days'
	),
	(
		'44444444-4444-4444-4444-444444444211',
		'22222222-2222-2222-2222-222222222211',
		NULL,
		'Kirill',
		5,
		'Handy kit for travel workouts. Resistance levels feel balanced and the pouch is compact.',
		TRUE,
		NOW() - INTERVAL '3 days'
	),
	(
		'44444444-4444-4444-4444-444444444212',
		'22222222-2222-2222-2222-222222222212',
		NULL,
		'Marina',
		4,
		'Pleasant scents and the ceramic jars look good even after the candles are finished.',
		TRUE,
		NOW() - INTERVAL '2 days'
	),
	(
		'44444444-4444-4444-4444-444444444213',
		'22222222-2222-2222-2222-222222222213',
		NULL,
		'Viktor',
		4,
		'Simple but effective. The weighted base keeps cables from sliding off the desk.',
		TRUE,
		NOW() - INTERVAL '2 days'
	),
	(
		'44444444-4444-4444-4444-444444444214',
		'22222222-2222-2222-2222-222222222214',
		NULL,
		'Yulia',
		5,
		'Paired quickly with the phone and the loud alert makes keys easy to find.',
		TRUE,
		NOW() - INTERVAL '1 day'
	),
	(
		'44444444-4444-4444-4444-444444444215',
		'22222222-2222-2222-2222-222222222215',
		NULL,
		'Sergey',
		5,
		'Keeps water cold for most of the day and fits nicely in a backpack side pocket.',
		TRUE,
		NOW() - INTERVAL '1 day'
	)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_product_reviews_updated_at ON product_reviews;
DROP INDEX IF EXISTS idx_product_reviews_user_product_unique;
DROP INDEX IF EXISTS idx_product_reviews_product_created_at;
DROP TABLE IF EXISTS product_reviews;
-- +goose StatementEnd
