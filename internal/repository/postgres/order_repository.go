package postgres

import (
	"context"
	"fmt"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type orderScanner interface {
	Scan(dest ...any) error
}

type orderQueryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type checkoutCartRow struct {
	ProductID uuid.UUID
	Name      string
	SKU       string
	UnitPrice float64
	Currency  string
	StockQty  int
	IsActive  bool
	Quantity  int
}

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Checkout(ctx context.Context, userID uuid.UUID, place domain.Place) (domain.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Order{}, fmt.Errorf("begin order checkout transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	rows, err := tx.Query(ctx, `
		SELECT p.id, p.name, p.sku, p.price::double precision, p.currency,
		       p.stock_qty, p.is_active, ci.quantity
		FROM cart_items ci
		INNER JOIN products p ON p.id = ci.product_id
		WHERE ci.user_id = $1
		ORDER BY ci.created_at ASC
		FOR UPDATE OF ci, p
	`, userID)
	if err != nil {
		return domain.Order{}, mapError(err)
	}

	cartRows := make([]checkoutCartRow, 0)
	for rows.Next() {
		var item checkoutCartRow
		if err := rows.Scan(
			&item.ProductID,
			&item.Name,
			&item.SKU,
			&item.UnitPrice,
			&item.Currency,
			&item.StockQty,
			&item.IsActive,
			&item.Quantity,
		); err != nil {
			rows.Close()
			return domain.Order{}, mapError(err)
		}
		cartRows = append(cartRows, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return domain.Order{}, mapError(err)
	}
	rows.Close()

	if len(cartRows) == 0 {
		return domain.Order{}, domain.ErrCartEmpty
	}

	currency := cartRows[0].Currency
	total := 0.0
	items := make([]domain.OrderItem, 0, len(cartRows))

	for _, item := range cartRows {
		if !item.IsActive {
			return domain.Order{}, domain.ErrUnavailable
		}
		if item.StockQty < item.Quantity {
			return domain.Order{}, domain.ErrStockShortage
		}
		if item.Currency != currency {
			return domain.Order{}, domain.ErrConflict
		}

		lineTotal := item.UnitPrice * float64(item.Quantity)
		items = append(items, domain.OrderItem{
			ID:          uuid.New(),
			ProductID:   item.ProductID,
			ProductName: item.Name,
			SKU:         item.SKU,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			LineTotal:   lineTotal,
			Currency:    item.Currency,
		})
		total += lineTotal
	}

	order := domain.Order{
		ID:          uuid.New(),
		UserID:      userID,
		PlaceID:     place.ID,
		PlaceTitle:  place.Title,
		AddressText: place.AddressText,
		Lat:         place.Lat,
		Lon:         place.Lon,
		Status:      domain.OrderStatusPending,
		Currency:    currency,
		TotalAmount: total,
		Items:       items,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO orders (id, user_id, place_id, place_title, address_text, lat, lon, status, currency, total_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`, order.ID, order.UserID, order.PlaceID, order.PlaceTitle, order.AddressText, order.Lat, order.Lon, order.Status, order.Currency, order.TotalAmount).Scan(
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		return domain.Order{}, mapError(err)
	}

	for index := range items {
		items[index].OrderID = order.ID
		err = tx.QueryRow(ctx, `
			INSERT INTO order_items (id, order_id, product_id, product_name, sku, unit_price, quantity, line_total, currency)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING created_at
		`, items[index].ID, order.ID, items[index].ProductID, items[index].ProductName, items[index].SKU, items[index].UnitPrice, items[index].Quantity, items[index].LineTotal, items[index].Currency).Scan(
			&items[index].CreatedAt,
		)
		if err != nil {
			return domain.Order{}, mapError(err)
		}

		_, err = tx.Exec(ctx, `
			UPDATE products
			SET stock_qty = stock_qty - $2
			WHERE id = $1
		`, items[index].ProductID, items[index].Quantity)
		if err != nil {
			return domain.Order{}, mapError(err)
		}
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM cart_items
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return domain.Order{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Order{}, fmt.Errorf("commit order checkout transaction: %w", err)
	}

	order.Items = items
	return order, nil
}

func (r *OrderRepository) ListByUser(ctx context.Context, userID uuid.UUID, page, limit int) (domain.PageResult[domain.Order], error) {
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM orders
		WHERE user_id = $1
	`, userID).Scan(&total); err != nil {
		return domain.PageResult[domain.Order]{}, mapError(err)
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, place_id, place_title, address_text, lat, lon,
		       status, currency, total_amount::double precision, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return domain.PageResult[domain.Order]{}, mapError(err)
	}
	defer rows.Close()

	items := make([]domain.Order, 0, limit)
	orderIDs := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var order domain.Order
		if err := scanOrder(rows, &order); err != nil {
			return domain.PageResult[domain.Order]{}, mapError(err)
		}
		items = append(items, order)
		orderIDs = append(orderIDs, order.ID)
	}
	if err := rows.Err(); err != nil {
		return domain.PageResult[domain.Order]{}, mapError(err)
	}

	itemsByOrderID, err := r.loadOrderItems(ctx, r.db, orderIDs)
	if err != nil {
		return domain.PageResult[domain.Order]{}, err
	}
	for index := range items {
		items[index].Items = itemsByOrderID[items[index].ID]
	}

	return domain.PageResult[domain.Order]{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil
}

func (r *OrderRepository) GetByIDForUser(ctx context.Context, orderID, userID uuid.UUID) (domain.Order, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, place_id, place_title, address_text, lat, lon,
		       status, currency, total_amount::double precision, created_at, updated_at
		FROM orders
		WHERE id = $1 AND user_id = $2
	`, orderID, userID)

	var order domain.Order
	if err := scanOrder(row, &order); err != nil {
		return domain.Order{}, mapError(err)
	}

	itemsByOrderID, err := r.loadOrderItems(ctx, r.db, []uuid.UUID{order.ID})
	if err != nil {
		return domain.Order{}, err
	}
	order.Items = itemsByOrderID[order.ID]
	return order, nil
}

func (r *OrderRepository) loadOrderItems(ctx context.Context, db orderQueryer, orderIDs []uuid.UUID) (map[uuid.UUID][]domain.OrderItem, error) {
	itemsByOrderID := make(map[uuid.UUID][]domain.OrderItem, len(orderIDs))
	if len(orderIDs) == 0 {
		return itemsByOrderID, nil
	}

	rows, err := db.Query(ctx, `
		SELECT id, order_id, product_id, product_name, sku, unit_price::double precision,
		       quantity, line_total::double precision, currency, created_at
		FROM order_items
		WHERE order_id = ANY($1)
		ORDER BY created_at ASC, id ASC
	`, orderIDs)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.ProductName,
			&item.SKU,
			&item.UnitPrice,
			&item.Quantity,
			&item.LineTotal,
			&item.Currency,
			&item.CreatedAt,
		); err != nil {
			return nil, mapError(err)
		}
		itemsByOrderID[item.OrderID] = append(itemsByOrderID[item.OrderID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}
	return itemsByOrderID, nil
}

func scanOrder(row orderScanner, order *domain.Order) error {
	return row.Scan(
		&order.ID,
		&order.UserID,
		&order.PlaceID,
		&order.PlaceTitle,
		&order.AddressText,
		&order.Lat,
		&order.Lon,
		&order.Status,
		&order.Currency,
		&order.TotalAmount,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
}
