package handlers

import (
	"context"
	"net/http"
	"strings"

	"marketplace-backend/internal/domain"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type FavoritesService interface {
	Add(ctx context.Context, userID, productID uuid.UUID) (bool, error)
	Remove(ctx context.Context, userID, productID uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID, page, limit int) (domain.PageResult[domain.Product], error)
}

type FavoritesHandler struct {
	service FavoritesService
}

func NewFavoritesHandler(service FavoritesService) *FavoritesHandler {
	return &FavoritesHandler{service: service}
}

func (h *FavoritesHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	productID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "product_id")))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	created, err := h.service.Add(r.Context(), userID, productID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"created": created})
}

func (h *FavoritesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	productID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "product_id")))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	if err := h.service.Remove(r.Context(), userID, productID); err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *FavoritesHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	page := parseIntWithDefault(strings.TrimSpace(r.URL.Query().Get("page")), 1)
	limit := parseIntWithDefault(strings.TrimSpace(r.URL.Query().Get("limit")), 20)
	if page <= 0 || limit <= 0 {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	result, err := h.service.List(r.Context(), userID, page, limit)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
