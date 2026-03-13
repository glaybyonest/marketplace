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

type OrdersService interface {
	Checkout(ctx context.Context, userID, placeID uuid.UUID) (domain.Order, error)
	List(ctx context.Context, userID uuid.UUID, page, limit int) (domain.PageResult[domain.Order], error)
	GetByID(ctx context.Context, userID, orderID uuid.UUID) (domain.Order, error)
}

type OrdersHandler struct {
	service  OrdersService
	validate *validator.Validate
}

func NewOrdersHandler(service OrdersService) *OrdersHandler {
	return &OrdersHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *OrdersHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	var req dto.CheckoutRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	placeID, err := uuid.Parse(strings.TrimSpace(req.PlaceID))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	order, err := h.service.Checkout(r.Context(), userID, placeID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, order)
}

func (h *OrdersHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	page := parseIntWithDefault(r.URL.Query().Get("page"), 1)
	limit := parseIntWithDefault(r.URL.Query().Get("limit"), 20)

	result, err := h.service.List(r.Context(), userID, page, limit)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *OrdersHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	order, err := h.service.GetByID(r.Context(), userID, orderID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, order)
}
