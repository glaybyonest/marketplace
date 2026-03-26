-- +goose Up
-- +goose StatementBegin
INSERT INTO users (
	id,
	email,
	password_hash,
	full_name,
	role,
	email_verified_at,
	is_active
)
VALUES
	(
		'55555555-5555-5555-5555-555555555401',
		'anna.ivanova.reviewer@seed.marketplace.local',
		'$2a$10$5c005ew/Ph3.vc3qqANJ.uXIdIEP13UaZIUyQR1ztneu0xmt9S4fi',
		'Анна Иванова',
		'customer',
		NOW(),
		TRUE
	),
	(
		'55555555-5555-5555-5555-555555555402',
		'dmitriy.smirnov.reviewer@seed.marketplace.local',
		'$2a$10$5c005ew/Ph3.vc3qqANJ.uXIdIEP13UaZIUyQR1ztneu0xmt9S4fi',
		'Дмитрий Смирнов',
		'customer',
		NOW(),
		TRUE
	),
	(
		'55555555-5555-5555-5555-555555555403',
		'elena.kuznetsova.reviewer@seed.marketplace.local',
		'$2a$10$5c005ew/Ph3.vc3qqANJ.uXIdIEP13UaZIUyQR1ztneu0xmt9S4fi',
		'Елена Кузнецова',
		'customer',
		NOW(),
		TRUE
	),
	(
		'55555555-5555-5555-5555-555555555404',
		'sergey.popov.reviewer@seed.marketplace.local',
		'$2a$10$5c005ew/Ph3.vc3qqANJ.uXIdIEP13UaZIUyQR1ztneu0xmt9S4fi',
		'Сергей Попов',
		'customer',
		NOW(),
		TRUE
	),
	(
		'55555555-5555-5555-5555-555555555405',
		'mariya.sokolova.reviewer@seed.marketplace.local',
		'$2a$10$5c005ew/Ph3.vc3qqANJ.uXIdIEP13UaZIUyQR1ztneu0xmt9S4fi',
		'Мария Соколова',
		'customer',
		NOW(),
		TRUE
	),
	(
		'55555555-5555-5555-5555-555555555406',
		'aleksey.petrov.reviewer@seed.marketplace.local',
		'$2a$10$5c005ew/Ph3.vc3qqANJ.uXIdIEP13UaZIUyQR1ztneu0xmt9S4fi',
		'Алексей Петров',
		'customer',
		NOW(),
		TRUE
	),
	(
		'55555555-5555-5555-5555-555555555407',
		'olga.vasileva.reviewer@seed.marketplace.local',
		'$2a$10$5c005ew/Ph3.vc3qqANJ.uXIdIEP13UaZIUyQR1ztneu0xmt9S4fi',
		'Ольга Васильева',
		'customer',
		NOW(),
		TRUE
	),
	(
		'55555555-5555-5555-5555-555555555408',
		'irina.morozova.reviewer@seed.marketplace.local',
		'$2a$10$5c005ew/Ph3.vc3qqANJ.uXIdIEP13UaZIUyQR1ztneu0xmt9S4fi',
		'Ирина Морозова',
		'customer',
		NOW(),
		TRUE
	),
	(
		'55555555-5555-5555-5555-555555555409',
		'pavel.novikov.reviewer@seed.marketplace.local',
		'$2a$10$5c005ew/Ph3.vc3qqANJ.uXIdIEP13UaZIUyQR1ztneu0xmt9S4fi',
		'Павел Новиков',
		'customer',
		NOW(),
		TRUE
	),
	(
		'55555555-5555-5555-5555-555555555410',
		'natalya.volkova.reviewer@seed.marketplace.local',
		'$2a$10$5c005ew/Ph3.vc3qqANJ.uXIdIEP13UaZIUyQR1ztneu0xmt9S4fi',
		'Наталья Волкова',
		'customer',
		NOW(),
		TRUE
	)
ON CONFLICT (id) DO UPDATE
SET
	email = EXCLUDED.email,
	password_hash = EXCLUDED.password_hash,
	full_name = EXCLUDED.full_name,
	role = EXCLUDED.role,
	email_verified_at = EXCLUDED.email_verified_at,
	is_active = EXCLUDED.is_active;

UPDATE product_reviews
SET is_published = FALSE;

DELETE FROM product_reviews
WHERE user_id IN (
	'55555555-5555-5555-5555-555555555401',
	'55555555-5555-5555-5555-555555555402',
	'55555555-5555-5555-5555-555555555403',
	'55555555-5555-5555-5555-555555555404',
	'55555555-5555-5555-5555-555555555405',
	'55555555-5555-5555-5555-555555555406',
	'55555555-5555-5555-5555-555555555407',
	'55555555-5555-5555-5555-555555555408',
	'55555555-5555-5555-5555-555555555409',
	'55555555-5555-5555-5555-555555555410'
);

WITH seed_users AS (
	SELECT *
	FROM (
		VALUES
			(1, '55555555-5555-5555-5555-555555555401'::uuid, 'Анна Иванова'),
			(2, '55555555-5555-5555-5555-555555555402'::uuid, 'Дмитрий Смирнов'),
			(3, '55555555-5555-5555-5555-555555555403'::uuid, 'Елена Кузнецова'),
			(4, '55555555-5555-5555-5555-555555555404'::uuid, 'Сергей Попов'),
			(5, '55555555-5555-5555-5555-555555555405'::uuid, 'Мария Соколова'),
			(6, '55555555-5555-5555-5555-555555555406'::uuid, 'Алексей Петров'),
			(7, '55555555-5555-5555-5555-555555555407'::uuid, 'Ольга Васильева'),
			(8, '55555555-5555-5555-5555-555555555408'::uuid, 'Ирина Морозова'),
			(9, '55555555-5555-5555-5555-555555555409'::uuid, 'Павел Новиков'),
			(10, '55555555-5555-5555-5555-555555555410'::uuid, 'Наталья Волкова')
	) AS users(slot, user_id, full_name)
),
seed_comments AS (
	SELECT *
	FROM (
		VALUES
			(1, 'Отличный товар, качество приятно удивило и пользоваться им действительно удобно.'),
			(2, 'Покупка полностью оправдала ожидания, всё сделано аккуратно и добротно.'),
			(3, 'Очень доволен заказом, товар выглядит отлично и в использовании радует каждый день.'),
			(4, 'Хорошее качество материалов, пользоваться приятно и всё работает как надо.'),
			(5, 'Приятно удивила продуманность деталей, товар оставляет очень хорошее впечатление.'),
			(6, 'Всё пришло в отличном состоянии, качество на высоком уровне и ощущается надёжность.'),
			(7, 'Товар удобный и практичный, вживую смотрится даже лучше, чем ожидалось.'),
			(8, 'Очень удачная покупка, всё аккуратно сделано и приятно использовать каждый день.'),
			(9, 'Качество отличное, товар полностью соответствует описанию и радует в работе.'),
			(10, 'Всё понравилось: внешний вид, удобство и общее впечатление от товара очень хорошее.')
	) AS comments(slot, comment)
),
products_ranked AS (
	SELECT
		p.id AS product_id,
		ROW_NUMBER() OVER (ORDER BY p.created_at, p.id) AS product_idx
	FROM products p
),
review_slots AS (
	SELECT generate_series(1, 5) AS review_slot
),
desired_reviews AS (
	SELECT
		p.product_id,
		p.product_idx,
		r.review_slot,
		((p.product_idx + r.review_slot - 2) % 10) + 1 AS user_slot,
		((p.product_idx * 3 + r.review_slot - 1) % 10) + 1 AS comment_slot,
		CASE
			WHEN MOD(p.product_idx + r.review_slot, 4) = 0 THEN 4
			ELSE 5
		END AS rating,
		NOW()
			- ((r.review_slot - 1) * INTERVAL '18 hours')
			- (MOD(p.product_idx, 7) * INTERVAL '3 hours')
			- (MOD(p.product_idx, 5) * INTERVAL '1 day') AS created_at
	FROM products_ranked p
	CROSS JOIN review_slots r
)
INSERT INTO product_reviews (
	product_id,
	user_id,
	author_name,
	rating,
	comment,
	is_published,
	created_at
)
SELECT
	dr.product_id,
	su.user_id,
	su.full_name,
	dr.rating,
	sc.comment,
	TRUE,
	dr.created_at
FROM desired_reviews dr
INNER JOIN seed_users su ON su.slot = dr.user_slot
INNER JOIN seed_comments sc ON sc.slot = dr.comment_slot;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM product_reviews
WHERE user_id IN (
	'55555555-5555-5555-5555-555555555401',
	'55555555-5555-5555-5555-555555555402',
	'55555555-5555-5555-5555-555555555403',
	'55555555-5555-5555-5555-555555555404',
	'55555555-5555-5555-5555-555555555405',
	'55555555-5555-5555-5555-555555555406',
	'55555555-5555-5555-5555-555555555407',
	'55555555-5555-5555-5555-555555555408',
	'55555555-5555-5555-5555-555555555409',
	'55555555-5555-5555-5555-555555555410'
);

DELETE FROM users
WHERE id IN (
	'55555555-5555-5555-5555-555555555401',
	'55555555-5555-5555-5555-555555555402',
	'55555555-5555-5555-5555-555555555403',
	'55555555-5555-5555-5555-555555555404',
	'55555555-5555-5555-5555-555555555405',
	'55555555-5555-5555-5555-555555555406',
	'55555555-5555-5555-5555-555555555407',
	'55555555-5555-5555-5555-555555555408',
	'55555555-5555-5555-5555-555555555409',
	'55555555-5555-5555-5555-555555555410'
);

UPDATE product_reviews
SET is_published = TRUE;
-- +goose StatementEnd
