package domain

import (
	"time"

	"github.com/google/uuid"
)

type CartItem struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	SKU       string    `json:"sku"`
	Price     float64   `json:"unit_price"`
	Quantity  int       `json:"quantity"`
	LineTotal float64   `json:"line_total"`
	Currency  string    `json:"currency"`
	StockQty  int       `json:"stock_qty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Cart struct {
	Items       []CartItem `json:"items"`
	TotalAmount float64    `json:"total_amount"`
	Currency    string     `json:"currency"`
	TotalItems  int        `json:"total_items"`
}

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type OrderItem struct {
	ID          uuid.UUID `json:"id"`
	OrderID     uuid.UUID `json:"order_id"`
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	SKU         string    `json:"sku"`
	UnitPrice   float64   `json:"unit_price"`
	Quantity    int       `json:"quantity"`
	LineTotal   float64   `json:"line_total"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
}

type Order struct {
	ID          uuid.UUID   `json:"id"`
	UserID      uuid.UUID   `json:"user_id"`
	PlaceID     uuid.UUID   `json:"place_id"`
	PlaceTitle  string      `json:"place_title"`
	AddressText string      `json:"address_text"`
	Lat         *float64    `json:"lat,omitempty"`
	Lon         *float64    `json:"lon,omitempty"`
	Status      OrderStatus `json:"status"`
	Currency    string      `json:"currency"`
	TotalAmount float64     `json:"total_amount"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Items       []OrderItem `json:"items"`
}
