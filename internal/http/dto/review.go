package dto

type CreateProductReviewRequest struct {
	Rating  int    `json:"rating" validate:"required,min=1,max=5"`
	Comment string `json:"comment" validate:"required,min=3,max=1500"`
}
