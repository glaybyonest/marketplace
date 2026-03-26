package domain

import (
	"time"

	"github.com/google/uuid"
)

// Category represents a product category in adjacency-list form.
type Category struct {
	ID        uuid.UUID  `json:"id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CategoryNode is nested category tree response.
type CategoryNode struct {
	ID        uuid.UUID      `json:"id"`
	ParentID  *uuid.UUID     `json:"parent_id,omitempty"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Children  []CategoryNode `json:"children"`
}

// Product represents an item in the catalog.
type Product struct {
	ID           uuid.UUID      `json:"id"`
	CategoryID   uuid.UUID      `json:"category_id"`
	CategoryName string         `json:"category_name,omitempty"`
	SellerID     *uuid.UUID     `json:"seller_id,omitempty"`
	SellerName   string         `json:"seller_name,omitempty"`
	SellerSlug   string         `json:"seller_slug,omitempty"`
	Name         string         `json:"name"`
	Slug         string         `json:"slug"`
	Description  string         `json:"description,omitempty"`
	Price        float64        `json:"price"`
	Currency     string         `json:"currency"`
	SKU          string         `json:"sku"`
	ImageURL     string         `json:"image_url,omitempty"`
	Images       []string       `json:"images"`
	Brand        string         `json:"brand,omitempty"`
	Unit         string         `json:"unit,omitempty"`
	Specs        map[string]any `json:"specs,omitempty"`
	Rating       float64        `json:"rating"`
	ReviewsCount int            `json:"reviews_count"`
	StockQty     int            `json:"stock_qty"`
	IsActive     bool           `json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type Review struct {
	ID        uuid.UUID  `json:"id"`
	ProductID uuid.UUID  `json:"product_id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	UserName  string     `json:"user_name"`
	Rating    int        `json:"rating"`
	Comment   string     `json:"comment"`
	CreatedAt time.Time  `json:"created_at"`
}

// ProductFilter describes catalog list query.
type ProductFilter struct {
	CategoryID      *uuid.UUID
	SellerID        *uuid.UUID
	Query           string
	MinPrice        *float64
	MaxPrice        *float64
	InStock         *bool
	IsActive        *bool
	IncludeInactive bool
	Page            int
	Limit           int
	Sort            string
}

type SearchSuggestion struct {
	Text string `json:"text"`
	Kind string `json:"kind"`
}

type PopularSearch struct {
	Query       string `json:"query"`
	SearchCount int    `json:"search_count"`
}

const (
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
	SortNew       = "new"
)

const (
	ProductEventView        = "view"
	ProductEventFavoriteAdd = "favorite_add"
)
