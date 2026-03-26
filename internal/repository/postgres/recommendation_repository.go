package postgres

import (
	"context"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/usecase"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecommendationRepository struct {
	db *pgxpool.Pool
}

func NewRecommendationRepository(db *pgxpool.Pool) *RecommendationRepository {
	return &RecommendationRepository{db: db}
}

func (r *RecommendationRepository) FavoriteProductIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
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

func (r *RecommendationRepository) FavoriteCategoryIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
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

func (r *RecommendationRepository) ListByCategories(
	ctx context.Context,
	categoryIDs []uuid.UUID,
	excludeProductIDs []uuid.UUID,
	limit int,
) ([]domain.Product, error) {
	if len(categoryIDs) == 0 {
		return []domain.Product{}, nil
	}

	var exclude any
	if len(excludeProductIDs) > 0 {
		exclude = excludeProductIDs
	}

	rows, err := r.db.Query(ctx, `
		SELECT `+productSelectColumns+`
		FROM products p
		`+productSelectJoins+`
		LEFT JOIN product_popularity_stats s ON s.product_id = p.id
		WHERE p.is_active = TRUE
		  AND p.category_id = ANY($1)
		  AND ($2::uuid[] IS NULL OR NOT (p.id = ANY($2)))
		ORDER BY COALESCE(s.score, 0) DESC, COALESCE(s.last_event_at, p.created_at) DESC, p.created_at DESC
		LIMIT $3
	`, categoryIDs, exclude, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.Product, 0, limit)
	for rows.Next() {
		var product domain.Product
		if err := scanProduct(rows, &product); err != nil {
			return nil, mapError(err)
		}
		items = append(items, product)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

func (r *RecommendationRepository) ListPopular(ctx context.Context, excludeProductIDs []uuid.UUID, limit int) ([]domain.Product, error) {
	var exclude any
	if len(excludeProductIDs) > 0 {
		exclude = excludeProductIDs
	}

	rows, err := r.db.Query(ctx, `
		SELECT `+productSelectColumns+`
		FROM products p
		`+productSelectJoins+`
		LEFT JOIN product_popularity_stats s ON s.product_id = p.id
		WHERE p.is_active = TRUE
		  AND ($1::uuid[] IS NULL OR NOT (p.id = ANY($1)))
		ORDER BY COALESCE(s.score, 0) DESC, COALESCE(s.last_event_at, p.created_at) DESC, p.created_at DESC
		LIMIT $2
	`, exclude, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.Product, 0, limit)
	for rows.Next() {
		var product domain.Product
		if err := scanProduct(rows, &product); err != nil {
			return nil, mapError(err)
		}
		items = append(items, product)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

func (r *RecommendationRepository) ListCached(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Product, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+productSelectColumns+`
		FROM user_recommendations ur
		INNER JOIN products p ON p.id = ur.product_id
		`+productSelectJoins+`
		WHERE ur.user_id = $1
		  AND p.is_active = TRUE
		ORDER BY ur.rank ASC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.Product, 0, limit)
	for rows.Next() {
		var product domain.Product
		if err := scanProduct(rows, &product); err != nil {
			return nil, mapError(err)
		}
		items = append(items, product)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

func (r *RecommendationRepository) ReplaceCached(
	ctx context.Context,
	userID uuid.UUID,
	items []usecase.RecommendationCacheItem,
	refreshedAt time.Time,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM user_recommendations WHERE user_id = $1`, userID); err != nil {
		return mapError(err)
	}

	for _, item := range items {
		if item.ProductID == uuid.Nil || item.Rank <= 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_recommendations (user_id, product_id, rank, source, refreshed_at)
			VALUES ($1, $2, $3, $4, $5)
		`, userID, item.ProductID, item.Rank, item.Source, refreshedAt); err != nil {
			return mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (r *RecommendationRepository) ActiveUserIDs(ctx context.Context, since time.Time, limit int) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		WITH activity AS (
			SELECT f.user_id, MAX(f.created_at) AS activity_at
			FROM favorites f
			GROUP BY f.user_id
			UNION ALL
			SELECT e.user_id, MAX(e.created_at) AS activity_at
			FROM user_product_events e
			GROUP BY e.user_id
			UNION ALL
			SELECT o.user_id, MAX(o.created_at) AS activity_at
			FROM orders o
			GROUP BY o.user_id
		)
		SELECT a.user_id
		FROM activity a
		INNER JOIN users u ON u.id = a.user_id
		WHERE u.is_active = TRUE
		  AND a.activity_at >= $1
		GROUP BY a.user_id
		ORDER BY MAX(a.activity_at) DESC
		LIMIT $2
	`, since, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	userIDs := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, mapError(err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return userIDs, nil
}

func (r *RecommendationRepository) RefreshPopularityStats(ctx context.Context, refreshedAt time.Time) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM product_popularity_stats`); err != nil {
		return 0, mapError(err)
	}

	cmd, err := tx.Exec(ctx, `
		INSERT INTO product_popularity_stats (
			product_id,
			favorite_count,
			favorite_event_count,
			view_count,
			score,
			last_event_at,
			refreshed_at
		)
		WITH favorite_stats AS (
			SELECT product_id, COUNT(*)::int AS favorite_count, MAX(created_at) AS last_favorite_at
			FROM favorites
			GROUP BY product_id
		),
		event_stats AS (
			SELECT
				product_id,
				COUNT(*) FILTER (WHERE event_type = 'favorite_add')::int AS favorite_event_count,
				COUNT(*) FILTER (WHERE event_type = 'view')::int AS view_count,
				MAX(created_at) AS last_event_at
			FROM user_product_events
			GROUP BY product_id
		)
		SELECT
			p.id,
			COALESCE(fs.favorite_count, 0),
			COALESCE(es.favorite_event_count, 0),
			COALESCE(es.view_count, 0),
			(COALESCE(fs.favorite_count, 0) * 5.0) + (COALESCE(es.favorite_event_count, 0) * 3.0) + COALESCE(es.view_count, 0),
			GREATEST(
				COALESCE(fs.last_favorite_at, p.created_at),
				COALESCE(es.last_event_at, p.created_at)
			),
			$1
		FROM products p
		LEFT JOIN favorite_stats fs ON fs.product_id = p.id
		LEFT JOIN event_stats es ON es.product_id = p.id
	`, refreshedAt)
	if err != nil {
		return 0, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}
