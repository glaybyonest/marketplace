package handlers

import (
	"context"
	"net/http"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/dto"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/http/response"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type ProfileService interface {
	Get(ctx context.Context, userID uuid.UUID) (domain.User, error)
	Update(ctx context.Context, userID uuid.UUID, fullName *string) (domain.User, error)
}

type ProfileHandler struct {
	service  ProfileService
	validate *validator.Validate
}

func NewProfileHandler(service ProfileService) *ProfileHandler {
	return &ProfileHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	user, err := h.service.Get(r.Context(), userID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	var req dto.UpdateProfileRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	user, err := h.service.Update(r.Context(), userID, req.FullName)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, user)
}
