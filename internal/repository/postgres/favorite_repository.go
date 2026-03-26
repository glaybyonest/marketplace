package postgres

import (
	"context"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FavoriteRepository struct {
	db *pgxpool.Pool
}

func NewFavoriteRepository(db *pgxpool.Pool) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

func (r *FavoriteRepository) Add(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	cmd, err := r.db.Exec(ctx, `
		INSERT INTO favorites (user_id, product_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, product_id) DO NOTHING
	`, userID, productID)
	if err != nil {
		return false, mapError(err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *FavoriteRepository) Remove(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	cmd, err := r.db.Exec(ctx, `
		DELETE FROM favorites
		WHERE user_id = $1 AND product_id = $2
	`, userID, productID)
	if err != nil {
		return false, mapError(err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *FavoriteRepository) List(ctx context.Context, userID uuid.UUID, page, limit int) (domain.PageResult[domain.Product], error) {
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM favorites f
		INNER JOIN products p ON p.id = f.product_id
		WHERE f.user_id = $1 AND p.is_active = TRUE
	`, userID).Scan(&total); err != nil {
		return domain.PageResult[domain.Product]{}, mapError(err)
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx, `
		SELECT `+productSelectColumns+`
		FROM favorites f
		INNER JOIN products p ON p.id = f.product_id
		`+productSelectJoins+`
		WHERE f.user_id = $1 AND p.is_active = TRUE
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return domain.PageResult[domain.Product]{}, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.Product, 0, limit)
	for rows.Next() {
		var product domain.Product
		if err := scanProduct(rows, &product); err != nil {
			return domain.PageResult[domain.Product]{}, mapError(err)
		}
		items = append(items, product)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.Product]{}, mapError(err)
	}

	return domain.PageResult[domain.Product]{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil
}

func (r *FavoriteRepository) FavoriteProductIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT product_id
		FROM favorites
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapError(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return ids, nil
}

func (r *FavoriteRepository) FavoriteCategoryIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT p.category_id
		FROM favorites f
		INNER JOIN products p ON p.id = f.product_id
		WHERE f.user_id = $1
		ORDER BY p.category_id
	`, userID)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapError(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return ids, nil
}
