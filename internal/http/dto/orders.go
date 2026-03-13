package dto

type CheckoutRequest struct {
	PlaceID string `json:"place_id" validate:"required"`
}
