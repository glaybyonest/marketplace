package handlers

import (
	"context"
	"net/http"
	"strings"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/dto"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/http/response"
	"marketplace-backend/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type PlacesService interface {
	Create(ctx context.Context, userID uuid.UUID, input usecase.CreatePlaceInput) (domain.Place, error)
	List(ctx context.Context, userID uuid.UUID) ([]domain.Place, error)
	Update(ctx context.Context, userID, placeID uuid.UUID, patch domain.PlacePatch) (domain.Place, error)
	Delete(ctx context.Context, userID, placeID uuid.UUID) error
}

type PlacesHandler struct {
	service  PlacesService
	validate *validator.Validate
}

func NewPlacesHandler(service PlacesService) *PlacesHandler {
	return &PlacesHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *PlacesHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	var req dto.CreatePlaceRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	item, err := h.service.Create(r.Context(), userID, usecase.CreatePlaceInput{
		Title:       req.Title,
		AddressText: req.AddressText,
		Lat:         req.Lat,
		Lon:         req.Lon,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, item)
}

func (h *PlacesHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	items, err := h.service.List(r.Context(), userID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *PlacesHandler) Patch(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	placeID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	var req dto.UpdatePlaceRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	item, err := h.service.Update(r.Context(), userID, placeID, domain.PlacePatch{
		Title:       req.Title,
		AddressText: req.AddressText,
		Lat:         req.Lat,
		Lon:         req.Lon,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, item)
}

func (h *PlacesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	placeID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	if err := h.service.Delete(r.Context(), userID, placeID); err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
