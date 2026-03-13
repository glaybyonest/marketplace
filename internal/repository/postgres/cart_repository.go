package postgres

import (
	"context"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type cartScanner interface {
	Scan(dest ...any) error
}

type CartRepository struct {
	db *pgxpool.Pool
}

func NewCartRepository(db *pgxpool.Pool) *CartRepository {
	return &CartRepository{db: db}
}

func (r *CartRepository) Get(ctx context.Context, userID uuid.UUID) (domain.Cart, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.id, p.name, p.slug, p.sku, p.price::double precision,
		       ci.quantity, (p.price * ci.quantity)::double precision, p.currency,
		       p.stock_qty, p.is_active, ci.created_at, ci.updated_at
		FROM cart_items ci
		INNER JOIN products p ON p.id = ci.product_id
		WHERE ci.user_id = $1
		ORDER BY ci.created_at DESC
	`, userID)
	if err != nil {
		return domain.Cart{}, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.CartItem, 0)
	totalAmount := 0.0
	totalItems := 0
	currency := "RUB"

	for rows.Next() {
		var item domain.CartItem
		if err := scanCartItem(rows, &item); err != nil {
			return domain.Cart{}, mapError(err)
		}
		if currency == "RUB" && item.Currency != "" {
			currency = item.Currency
		}
		totalAmount += item.LineTotal
		totalItems += item.Quantity
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Cart{}, mapError(err)
	}

	return domain.Cart{
		Items:       items,
		TotalAmount: totalAmount,
		Currency:    currency,
		TotalItems:  totalItems,
	}, nil
}

func (r *CartRepository) GetItem(ctx context.Context, userID, productID uuid.UUID) (domain.CartItem, error) {
	row := r.db.QueryRow(ctx, `
		SELECT p.id, p.id, p.name, p.slug, p.sku, p.price::double precision,
		       ci.quantity, (p.price * ci.quantity)::double precision, p.currency,
		       p.stock_qty, p.is_active, ci.created_at, ci.updated_at
		FROM cart_items ci
		INNER JOIN products p ON p.id = ci.product_id
		WHERE ci.user_id = $1 AND ci.product_id = $2
	`, userID, productID)

	var item domain.CartItem
	if err := scanCartItem(row, &item); err != nil {
		return domain.CartItem{}, mapError(err)
	}
	return item, nil
}

func (r *CartRepository) Add(ctx context.Context, userID, productID uuid.UUID, quantity int) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO cart_items (user_id, product_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, product_id)
		DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity, updated_at = NOW()
	`, userID, productID, quantity)
	return mapError(err)
}

func (r *CartRepository) SetQuantity(ctx context.Context, userID, productID uuid.UUID, quantity int) (bool, error) {
	cmd, err := r.db.Exec(ctx, `
		UPDATE cart_items
		SET quantity = $3, updated_at = NOW()
		WHERE user_id = $1 AND product_id = $2
	`, userID, productID, quantity)
	if err != nil {
		return false, mapError(err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *CartRepository) Delete(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	cmd, err := r.db.Exec(ctx, `
		DELETE FROM cart_items
		WHERE user_id = $1 AND product_id = $2
	`, userID, productID)
	if err != nil {
		return false, mapError(err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *CartRepository) Clear(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM cart_items
		WHERE user_id = $1
	`, userID)
	return mapError(err)
}

func scanCartItem(row cartScanner, item *domain.CartItem) error {
	return row.Scan(
		&item.ID,
		&item.ProductID,
		&item.Name,
		&item.Slug,
		&item.SKU,
		&item.Price,
		&item.Quantity,
		&item.LineTotal,
		&item.Currency,
		&item.StockQty,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
}
