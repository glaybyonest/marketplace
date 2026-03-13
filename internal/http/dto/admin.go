package dto

type AdminCategoryUpsertRequest struct {
	ParentID *string `json:"parent_id,omitempty" validate:"omitempty,uuid4"`
	Name     string  `json:"name" validate:"required,max=120"`
	Slug     string  `json:"slug" validate:"omitempty,max=120"`
}

type AdminProductUpsertRequest struct {
	CategoryID  string         `json:"category_id" validate:"required,uuid4"`
	Name        string         `json:"name" validate:"required,max=180"`
	Slug        string         `json:"slug" validate:"omitempty,max=180"`
	Description string         `json:"description" validate:"omitempty,max=5000"`
	Price       float64        `json:"price" validate:"gte=0"`
	Currency    string         `json:"currency" validate:"omitempty,len=3"`
	SKU         string         `json:"sku" validate:"required,max=120"`
	ImageURL    string         `json:"image_url" validate:"omitempty,url,max=2048"`
	Images      []string       `json:"images,omitempty"`
	Brand       string         `json:"brand" validate:"omitempty,max=120"`
	Unit        string         `json:"unit" validate:"omitempty,max=60"`
	Specs       map[string]any `json:"specs,omitempty"`
	StockQty    int            `json:"stock_qty" validate:"gte=0"`
	IsActive    *bool          `json:"is_active,omitempty"`
}

type AdminProductStockRequest struct {
	StockQty int `json:"stock_qty" validate:"gte=0"`
}
