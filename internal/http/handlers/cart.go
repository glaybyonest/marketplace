package handlers

import (
	"context"
	"net/http"
	"strings"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/dto"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type CartService interface {
	Get(ctx context.Context, userID uuid.UUID) (domain.Cart, error)
	AddItem(ctx context.Context, userID, productID uuid.UUID, quantity int) error
	UpdateItem(ctx context.Context, userID, productID uuid.UUID, quantity int) error
	DeleteItem(ctx context.Context, userID, productID uuid.UUID) error
	Clear(ctx context.Context, userID uuid.UUID) error
}

type CartHandler struct {
	service  CartService
	validate *validator.Validate
}

func NewCartHandler(service CartService) *CartHandler {
	return &CartHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *CartHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	cart, err := h.service.Get(r.Context(), userID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	var req dto.CartItemRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, err)
		return
	}

	productID, err := uuid.Parse(strings.TrimSpace(req.ProductID))
	if err != nil {
		writeDomainError(w, domain.ErrInvalidInput)
		return
	}

	if err := h.service.AddItem(r.Context(), userID, productID, req.Quantity); err != nil {
		writeDomainError(w, err)
		return
	}

	cart, err := h.service.Get(r.Context(), userID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, cart)
}

func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	productID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "product_id")))
	if err != nil {
		writeDomainError(w, domain.ErrInvalidInput)
		return
	}

	var req dto.UpdateCartItemRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, err)
		return
	}

	if err := h.service.UpdateItem(r.Context(), userID, productID, req.Quantity); err != nil {
		writeDomainError(w, err)
		return
	}

	cart, err := h.service.Get(r.Context(), userID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

func (h *CartHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	productID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "product_id")))
	if err != nil {
		writeDomainError(w, domain.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteItem(r.Context(), userID, productID); err != nil {
		writeDomainError(w, err)
		return
	}

	cart, err := h.service.Get(r.Context(), userID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

func (h *CartHandler) Clear(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, domain.ErrUnauthorized)
		return
	}

	if err := h.service.Clear(r.Context(), userID); err != nil {
		writeDomainError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, domain.Cart{
		Items:       []domain.CartItem{},
		TotalAmount: 0,
		Currency:    "RUB",
		TotalItems:  0,
	})
}
