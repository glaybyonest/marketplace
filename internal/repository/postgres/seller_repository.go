package postgres

import (
	"context"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/usecase"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SellerRepository struct {
	db *pgxpool.Pool
}

func NewSellerRepository(db *pgxpool.Pool) *SellerRepository {
	return &SellerRepository{db: db}
}

func (r *SellerRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (domain.SellerProfile, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			user_id,
			store_name,
			store_slug,
			COALESCE(legal_name, ''),
			COALESCE(description, ''),
			COALESCE(logo_url, ''),
			COALESCE(banner_url, ''),
			COALESCE(support_email, ''),
			COALESCE(support_phone, ''),
			COALESCE(city, ''),
			status,
			created_at,
			updated_at
		FROM seller_profiles
		WHERE user_id = $1
	`, userID)

	var profile domain.SellerProfile
	err := row.Scan(
		&profile.UserID,
		&profile.StoreName,
		&profile.StoreSlug,
		&profile.LegalName,
		&profile.Description,
		&profile.LogoURL,
		&profile.BannerURL,
		&profile.SupportEmail,
		&profile.SupportPhone,
		&profile.City,
		&profile.Status,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return domain.SellerProfile{}, mapError(err)
	}
	return profile, nil
}

func (r *SellerRepository) Upsert(ctx context.Context, input usecase.SellerProfileWriteInput) (domain.SellerProfile, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO seller_profiles (
			user_id,
			store_name,
			store_slug,
			legal_name,
			description,
			logo_url,
			banner_url,
			support_email,
			support_phone,
			city,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id)
		DO UPDATE SET
			store_name = EXCLUDED.store_name,
			store_slug = EXCLUDED.store_slug,
			legal_name = EXCLUDED.legal_name,
			description = EXCLUDED.description,
			logo_url = EXCLUDED.logo_url,
			banner_url = EXCLUDED.banner_url,
			support_email = EXCLUDED.support_email,
			support_phone = EXCLUDED.support_phone,
			city = EXCLUDED.city,
			status = EXCLUDED.status
		RETURNING
			user_id,
			store_name,
			store_slug,
			COALESCE(legal_name, ''),
			COALESCE(description, ''),
			COALESCE(logo_url, ''),
			COALESCE(banner_url, ''),
			COALESCE(support_email, ''),
			COALESCE(support_phone, ''),
			COALESCE(city, ''),
			status,
			created_at,
			updated_at
	`, input.UserID, input.StoreName, input.StoreSlug, input.LegalName, input.Description, input.LogoURL, input.BannerURL, input.SupportEmail, input.SupportPhone, input.City, input.Status)

	var profile domain.SellerProfile
	err := row.Scan(
		&profile.UserID,
		&profile.StoreName,
		&profile.StoreSlug,
		&profile.LegalName,
		&profile.Description,
		&profile.LogoURL,
		&profile.BannerURL,
		&profile.SupportEmail,
		&profile.SupportPhone,
		&profile.City,
		&profile.Status,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return domain.SellerProfile{}, mapError(err)
	}
	return profile, nil
}

func (r *SellerRepository) Metrics(ctx context.Context, userID uuid.UUID, lowStockThreshold int) (domain.SellerMetrics, error) {
	row := r.db.QueryRow(ctx, `
		WITH product_stats AS (
			SELECT
				COUNT(*)::int AS products_total,
				COUNT(*) FILTER (WHERE is_active = TRUE)::int AS active_products,
				COUNT(*) FILTER (WHERE is_active = FALSE)::int AS hidden_products,
				COUNT(*) FILTER (WHERE is_active = TRUE AND stock_qty <= $2)::int AS low_stock_products
			FROM products
			WHERE seller_id = $1
		),
		order_stats AS (
			SELECT
				COUNT(DISTINCT order_id)::int AS orders_total,
				COALESCE(SUM(quantity), 0)::int AS units_sold,
				COALESCE(SUM(line_total::double precision), 0) AS gross_revenue
			FROM order_items
			WHERE seller_id = $1
		)
		SELECT
			COALESCE(product_stats.products_total, 0),
			COALESCE(product_stats.active_products, 0),
			COALESCE(product_stats.hidden_products, 0),
			COALESCE(product_stats.low_stock_products, 0),
			COALESCE(order_stats.orders_total, 0),
			COALESCE(order_stats.units_sold, 0),
			COALESCE(order_stats.gross_revenue, 0)
		FROM product_stats, order_stats
	`, userID, lowStockThreshold)

	var metrics domain.SellerMetrics
	if err := row.Scan(
		&metrics.ProductsTotal,
		&metrics.ActiveProducts,
		&metrics.HiddenProducts,
		&metrics.LowStockProducts,
		&metrics.OrdersTotal,
		&metrics.UnitsSold,
		&metrics.GrossRevenue,
	); err != nil {
		return domain.SellerMetrics{}, mapError(err)
	}
	return metrics, nil
}

func (r *SellerRepository) ListLowStockProducts(ctx context.Context, userID uuid.UUID, threshold, limit int) ([]domain.Product, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+productSelectColumns+`
		FROM products p
		`+productSelectJoins+`
		WHERE p.seller_id = $1
		  AND p.is_active = TRUE
		  AND p.stock_qty <= $2
		ORDER BY p.stock_qty ASC, p.updated_at DESC
		LIMIT $3
	`, userID, threshold, limit)
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

func (r *SellerRepository) ListOrders(ctx context.Context, userID uuid.UUID, page, limit int) (domain.PageResult[domain.SellerOrderSummary], error) {
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM (
			SELECT DISTINCT order_id
			FROM order_items
			WHERE seller_id = $1
		) seller_orders
	`, userID).Scan(&total); err != nil {
		return domain.PageResult[domain.SellerOrderSummary]{}, mapError(err)
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx, `
		SELECT
			o.id,
			o.status,
			o.currency,
			o.created_at,
			o.place_title,
			COUNT(oi.id)::int AS items_count,
			COALESCE(SUM(oi.line_total::double precision), 0) AS gross_revenue
		FROM order_items oi
		INNER JOIN orders o ON o.id = oi.order_id
		WHERE oi.seller_id = $1
		GROUP BY o.id, o.status, o.currency, o.created_at, o.place_title
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return domain.PageResult[domain.SellerOrderSummary]{}, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.SellerOrderSummary, 0, limit)
	for rows.Next() {
		var item domain.SellerOrderSummary
		if err := rows.Scan(
			&item.OrderID,
			&item.Status,
			&item.Currency,
			&item.CreatedAt,
			&item.PlaceTitle,
			&item.ItemsCount,
			&item.GrossRevenue,
		); err != nil {
			return domain.PageResult[domain.SellerOrderSummary]{}, mapError(err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.SellerOrderSummary]{}, mapError(err)
	}

	return domain.PageResult[domain.SellerOrderSummary]{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil
}
