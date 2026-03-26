package postgres

import (
	"context"
	"fmt"
	"strings"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductReviewRepository struct {
	db *pgxpool.Pool
}

func NewProductReviewRepository(db *pgxpool.Pool) *ProductReviewRepository {
	return &ProductReviewRepository{db: db}
}

func (r *ProductReviewRepository) ListByProductID(ctx context.Context, productID uuid.UUID, limit int) ([]domain.Review, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			product_id,
			COALESCE(user_id::text, ''),
			author_name,
			rating,
			comment,
			created_at
		FROM product_reviews
		WHERE product_id = $1
			AND is_published = TRUE
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, productID, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.Review, 0, limit)
	for rows.Next() {
		var item domain.Review
		if err := scanReview(rows, &item); err != nil {
			return nil, mapError(err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}

	return items, nil
}

func (r *ProductReviewRepository) Create(
	ctx context.Context,
	productID, userID uuid.UUID,
	rating int,
	comment string,
) (domain.Review, error) {
	row := r.db.QueryRow(ctx, `
		WITH author AS (
			SELECT
				id,
				COALESCE(NULLIF(BTRIM(full_name), ''), split_part(email, '@', 1)) AS author_name
			FROM users
			WHERE id = $2
				AND is_active = TRUE
		),
		inserted AS (
			INSERT INTO product_reviews (
				product_id,
				user_id,
				author_name,
				rating,
				comment
			)
			SELECT
				$1,
				author.id,
				author.author_name,
				$3,
				$4
			FROM author
			RETURNING
				id,
				product_id,
				COALESCE(user_id::text, ''),
				author_name,
				rating,
				comment,
				created_at
		)
		SELECT *
		FROM inserted
	`, productID, userID, rating, strings.TrimSpace(comment))

	var item domain.Review
	if err := scanReview(row, &item); err != nil {
		return domain.Review{}, mapError(err)
	}

	return item, nil
}

func scanReview(row pgx.Row, review *domain.Review) error {
	var userIDText string

	if err := row.Scan(
		&review.ID,
		&review.ProductID,
		&userIDText,
		&review.UserName,
		&review.Rating,
		&review.Comment,
		&review.CreatedAt,
	); err != nil {
		return err
	}

	if strings.TrimSpace(userIDText) == "" {
		review.UserID = nil
		return nil
	}

	userID, err := uuid.Parse(userIDText)
	if err != nil {
		return fmt.Errorf("parse review user id: %w", err)
	}
	review.UserID = &userID

	return nil
}
