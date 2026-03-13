package dto

type CartItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,min=1,max=1000"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,min=1,max=1000"`
}
