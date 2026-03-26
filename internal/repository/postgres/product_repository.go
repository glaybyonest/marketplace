package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/usecase"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductRepository struct {
	db *pgxpool.Pool
}

const productSelectColumns = `
		p.id,
		p.category_id,
		c.name,
		COALESCE(p.seller_id::text, ''),
		COALESCE(sp.store_name, ''),
		COALESCE(sp.store_slug, ''),
		p.name,
		p.slug,
		COALESCE(p.description, ''),
		p.price::double precision,
		p.currency,
		p.sku,
		COALESCE(p.image_url, ''),
		COALESCE(p.gallery, '[]'::jsonb),
		COALESCE(p.brand, ''),
		COALESCE(p.unit, ''),
		COALESCE(p.specs, '{}'::jsonb),
		COALESCE(review_stats.average_rating, 0),
		COALESCE(review_stats.reviews_count, 0),
		p.stock_qty,
		p.is_active,
		p.created_at,
		p.updated_at
`

const productSelectJoins = `
		INNER JOIN categories c ON c.id = p.category_id
		LEFT JOIN seller_profiles sp ON sp.user_id = p.seller_id
		LEFT JOIN LATERAL (
			SELECT
				COALESCE(ROUND(AVG(pr.rating)::numeric, 1)::double precision, 0) AS average_rating,
				COUNT(*)::int AS reviews_count
			FROM product_reviews pr
			WHERE pr.product_id = p.id
				AND pr.is_published = TRUE
		) review_stats ON TRUE
`

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) List(ctx context.Context, filter domain.ProductFilter) (domain.PageResult[domain.Product], error) {
	where := make([]string, 0, 8)
	if !filter.IncludeInactive {
		where = append(where, "p.is_active = TRUE")
	}
	args := make([]any, 0, 8)

	if filter.CategoryID != nil {
		args = append(args, *filter.CategoryID)
		where = append(where, fmt.Sprintf("p.category_id = $%d", len(args)))
	}
	if filter.SellerID != nil {
		args = append(args, *filter.SellerID)
		where = append(where, fmt.Sprintf("p.seller_id = $%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+strings.ToLower(filter.Query)+"%")
		where = append(where, fmt.Sprintf(`(
			LOWER(p.name) LIKE $%d OR
			LOWER(COALESCE(p.description, '')) LIKE $%d OR
			LOWER(COALESCE(p.brand, '')) LIKE $%d OR
			LOWER(p.slug) LIKE $%d OR
			LOWER(p.sku) LIKE $%d OR
			LOWER(c.name) LIKE $%d OR
			LOWER(COALESCE(sp.store_name, '')) LIKE $%d
		)`, len(args), len(args), len(args), len(args), len(args), len(args), len(args)))
	}
	if filter.MinPrice != nil {
		args = append(args, *filter.MinPrice)
		where = append(where, fmt.Sprintf("p.price >= $%d", len(args)))
	}
	if filter.MaxPrice != nil {
		args = append(args, *filter.MaxPrice)
		where = append(where, fmt.Sprintf("p.price <= $%d", len(args)))
	}
	if filter.InStock != nil {
		if *filter.InStock {
			where = append(where, "p.stock_qty > 0")
		} else {
			where = append(where, "p.stock_qty >= 0")
		}
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		where = append(where, fmt.Sprintf("p.is_active = $%d", len(args)))
	}

	condition := "TRUE"
	if len(where) > 0 {
		condition = strings.Join(where, " AND ")
	}
	countSQL := `
		SELECT COUNT(*)
		FROM products p
		INNER JOIN categories c ON c.id = p.category_id
		LEFT JOIN seller_profiles sp ON sp.user_id = p.seller_id
		WHERE ` + condition

	var total int
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return domain.PageResult[domain.Product]{}, mapError(err)
	}

	orderBy := "p.created_at DESC"
	switch filter.Sort {
	case domain.SortPriceAsc:
		orderBy = "p.price ASC, p.created_at DESC"
	case domain.SortPriceDesc:
		orderBy = "p.price DESC, p.created_at DESC"
	case domain.SortNew:
		orderBy = "p.created_at DESC"
	}

	offset := (filter.Page - 1) * filter.Limit
	args = append(args, filter.Limit, offset)
	limitPos := len(args) - 1
	offsetPos := len(args)

	querySQL := `
		SELECT ` + productSelectColumns + `
		FROM products p
	` + productSelectJoins + `
		WHERE ` + condition + `
		ORDER BY ` + orderBy + `
		LIMIT $` + fmt.Sprint(limitPos) + ` OFFSET $` + fmt.Sprint(offsetPos)

	rows, err := r.db.Query(ctx, querySQL, args...)
	if err != nil {
		return domain.PageResult[domain.Product]{}, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.Product, 0, filter.Limit)
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
		Page:  filter.Page,
		Limit: filter.Limit,
		Total: total,
	}, nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	return r.getByCondition(ctx, "p.id = $1", id, false)
}

func (r *ProductRepository) GetBySlug(ctx context.Context, slug string) (domain.Product, error) {
	return r.getByCondition(ctx, "p.slug = $1", slug, false)
}

func (r *ProductRepository) GetByIDAny(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	return r.getByCondition(ctx, "p.id = $1", id, true)
}

func (r *ProductRepository) GetBySlugAny(ctx context.Context, slug string) (domain.Product, error) {
	return r.getByCondition(ctx, "p.slug = $1", slug, true)
}

func (r *ProductRepository) SearchSuggestions(ctx context.Context, query string, limit int) ([]domain.SearchSuggestion, error) {
	rows, err := r.db.Query(ctx, `
		WITH suggestions AS (
			SELECT DISTINCT p.name AS text, 'product' AS kind,
				CASE WHEN LOWER(p.name) LIKE $1 || '%' THEN 1 ELSE 4 END AS rank
			FROM products p
			INNER JOIN categories c ON c.id = p.category_id
			WHERE p.is_active = TRUE
			  AND (
				LOWER(p.name) LIKE '%' || $1 || '%'
				OR LOWER(COALESCE(p.description, '')) LIKE '%' || $1 || '%'
				OR LOWER(COALESCE(p.brand, '')) LIKE '%' || $1 || '%'
				OR LOWER(p.slug) LIKE '%' || $1 || '%'
				OR LOWER(p.sku) LIKE '%' || $1 || '%'
				OR LOWER(c.name) LIKE '%' || $1 || '%'
			  )
			UNION
			SELECT DISTINCT p.brand AS text, 'brand' AS kind,
				CASE WHEN LOWER(p.brand) LIKE $1 || '%' THEN 2 ELSE 5 END AS rank
			FROM products p
			WHERE p.is_active = TRUE
			  AND COALESCE(p.brand, '') <> ''
			  AND LOWER(p.brand) LIKE '%' || $1 || '%'
			UNION
			SELECT DISTINCT c.name AS text, 'category' AS kind,
				CASE WHEN LOWER(c.name) LIKE $1 || '%' THEN 3 ELSE 6 END AS rank
			FROM categories c
			INNER JOIN products p ON p.category_id = c.id
			WHERE p.is_active = TRUE
			  AND LOWER(c.name) LIKE '%' || $1 || '%'
		)
		SELECT text, kind
		FROM suggestions
		WHERE BTRIM(text) <> ''
		ORDER BY rank ASC, CHAR_LENGTH(text) ASC, text ASC
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.SearchSuggestion, 0, limit)
	for rows.Next() {
		var item domain.SearchSuggestion
		if err := rows.Scan(&item.Text, &item.Kind); err != nil {
			return nil, mapError(err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

func (r *ProductRepository) ListPopularSearches(ctx context.Context, limit int) ([]domain.PopularSearch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT query_text, search_count
		FROM search_queries
		ORDER BY search_count DESC, last_searched_at DESC, query_text ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.PopularSearch, 0, limit)
	for rows.Next() {
		var item domain.PopularSearch
		if err := rows.Scan(&item.Query, &item.SearchCount); err != nil {
			return nil, mapError(err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	if len(items) > 0 {
		return items, nil
	}

	fallbackRows, err := r.db.Query(ctx, `
		SELECT p.name, 0
		FROM products p
		WHERE p.is_active = TRUE
		ORDER BY p.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer fallbackRows.Close()

	fallback := make([]domain.PopularSearch, 0, limit)
	for fallbackRows.Next() {
		var item domain.PopularSearch
		if err := fallbackRows.Scan(&item.Query, &item.SearchCount); err != nil {
			return nil, mapError(err)
		}
		fallback = append(fallback, item)
	}
	if err := fallbackRows.Err(); err != nil {
		return nil, mapError(err)
	}
	return fallback, nil
}

func (r *ProductRepository) TrackSearchQuery(ctx context.Context, query string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO search_queries (query_text, search_count, last_searched_at)
		VALUES ($1, 1, NOW())
		ON CONFLICT (query_text)
		DO UPDATE SET
			search_count = search_queries.search_count + 1,
			last_searched_at = NOW()
	`, query)
	return mapError(err)
}

func (r *ProductRepository) Create(ctx context.Context, input usecase.ProductWriteInput) (domain.Product, error) {
	galleryRaw, specsRaw, err := marshalProductMetadata(input.Gallery, input.Specs)
	if err != nil {
		return domain.Product{}, err
	}

	var productID uuid.UUID
	if err := r.db.QueryRow(ctx, `
		INSERT INTO products (
			category_id,
			seller_id,
			name,
			slug,
			description,
			price,
			currency,
			sku,
			image_url,
			gallery,
			brand,
			unit,
			specs,
			stock_qty,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13::jsonb, $14, $15)
		RETURNING id
	`, input.CategoryID, input.SellerID, input.Name, input.Slug, input.Description, input.Price, input.Currency, input.SKU, input.ImageURL, galleryRaw, input.Brand, input.Unit, specsRaw, input.StockQty, input.IsActive).Scan(&productID); err != nil {
		return domain.Product{}, mapError(err)
	}
	return r.GetByIDAny(ctx, productID)
}

func (r *ProductRepository) Update(ctx context.Context, input usecase.ProductWriteInput) (domain.Product, error) {
	galleryRaw, specsRaw, err := marshalProductMetadata(input.Gallery, input.Specs)
	if err != nil {
		return domain.Product{}, err
	}

	var productID uuid.UUID
	if err := r.db.QueryRow(ctx, `
		UPDATE products p
		SET
			category_id = $2,
			seller_id = $3,
			name = $4,
			slug = $5,
			description = $6,
			price = $7,
			currency = $8,
			sku = $9,
			image_url = $10,
			gallery = $11::jsonb,
			brand = $12,
			unit = $13,
			specs = $14::jsonb,
			stock_qty = $15,
			is_active = $16
		WHERE p.id = $1
		RETURNING p.id
	`, input.ID, input.CategoryID, input.SellerID, input.Name, input.Slug, input.Description, input.Price, input.Currency, input.SKU, input.ImageURL, galleryRaw, input.Brand, input.Unit, specsRaw, input.StockQty, input.IsActive).Scan(&productID); err != nil {
		return domain.Product{}, mapError(err)
	}
	return r.GetByIDAny(ctx, productID)
}

func (r *ProductRepository) SetActive(ctx context.Context, id uuid.UUID, isActive bool) (domain.Product, error) {
	var productID uuid.UUID
	if err := r.db.QueryRow(ctx, `
		UPDATE products p
		SET is_active = $2
		WHERE p.id = $1
		RETURNING p.id
	`, id, isActive).Scan(&productID); err != nil {
		return domain.Product{}, mapError(err)
	}
	return r.GetByIDAny(ctx, productID)
}

func (r *ProductRepository) UpdateStock(ctx context.Context, id uuid.UUID, stockQty int) (domain.Product, error) {
	var productID uuid.UUID
	if err := r.db.QueryRow(ctx, `
		UPDATE products p
		SET stock_qty = $2
		WHERE p.id = $1
		RETURNING p.id
	`, id, stockQty).Scan(&productID); err != nil {
		return domain.Product{}, mapError(err)
	}
	return r.GetByIDAny(ctx, productID)
}

func (r *ProductRepository) getByCondition(ctx context.Context, condition string, arg any, includeInactive bool) (domain.Product, error) {
	where := condition
	if !includeInactive {
		where += " AND p.is_active = TRUE"
	}

	row := r.db.QueryRow(ctx, `
		SELECT `+productSelectColumns+`
		FROM products p
		`+productSelectJoins+`
		WHERE `+where, arg)

	var product domain.Product
	if err := scanProduct(row, &product); err != nil {
		return domain.Product{}, mapError(err)
	}
	return product, nil
}

func scanProduct(row pgx.Row, product *domain.Product) error {
	var galleryRaw []byte
	var specsRaw []byte
	var sellerIDText string

	err := row.Scan(
		&product.ID,
		&product.CategoryID,
		&product.CategoryName,
		&sellerIDText,
		&product.SellerName,
		&product.SellerSlug,
		&product.Name,
		&product.Slug,
		&product.Description,
		&product.Price,
		&product.Currency,
		&product.SKU,
		&product.ImageURL,
		&galleryRaw,
		&product.Brand,
		&product.Unit,
		&specsRaw,
		&product.Rating,
		&product.ReviewsCount,
		&product.StockQty,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if sellerIDText != "" {
		sellerID, err := uuid.Parse(sellerIDText)
		if err != nil {
			return fmt.Errorf("parse seller id: %w", err)
		}
		product.SellerID = &sellerID
	} else {
		product.SellerID = nil
	}

	images, err := decodeProductImages(product.ImageURL, galleryRaw)
	if err != nil {
		return err
	}
	product.Images = images

	specs, err := decodeProductSpecs(specsRaw)
	if err != nil {
		return err
	}
	product.Specs = specs

	return nil
}

func marshalProductMetadata(gallery []string, specs map[string]any) ([]byte, []byte, error) {
	normalizedGallery := make([]string, 0, len(gallery))
	for _, image := range gallery {
		image = strings.TrimSpace(image)
		if image == "" {
			continue
		}
		normalizedGallery = append(normalizedGallery, image)
	}

	galleryRaw, err := json.Marshal(normalizedGallery)
	if err != nil {
		return nil, nil, fmt.Errorf("encode product gallery: %w", err)
	}

	if specs == nil {
		specs = map[string]any{}
	}
	specsRaw, err := json.Marshal(specs)
	if err != nil {
		return nil, nil, fmt.Errorf("encode product specs: %w", err)
	}

	return galleryRaw, specsRaw, nil
}

func decodeProductImages(imageURL string, galleryRaw []byte) ([]string, error) {
	images := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)

	appendImage := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		images = append(images, value)
	}

	appendImage(imageURL)

	if len(galleryRaw) == 0 {
		return images, nil
	}

	var gallery []string
	if err := json.Unmarshal(galleryRaw, &gallery); err != nil {
		return nil, fmt.Errorf("decode product gallery: %w", err)
	}
	for _, image := range gallery {
		appendImage(image)
	}
	return images, nil
}

func decodeProductSpecs(specsRaw []byte) (map[string]any, error) {
	if len(specsRaw) == 0 {
		return map[string]any{}, nil
	}

	specs := make(map[string]any)
	if err := json.Unmarshal(specsRaw, &specs); err != nil {
		return nil, fmt.Errorf("decode product specs: %w", err)
	}
	return specs, nil
}
