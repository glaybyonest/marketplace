package handlers

import (
	"context"
	"net/http"
	"strings"

	"marketplace-backend/internal/domain"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/http/response"

	"github.com/google/uuid"
)

type RecommendationsService interface {
	Get(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Product, error)
}

type RecommendationsHandler struct {
	service RecommendationsService
}

func NewRecommendationsHandler(service RecommendationsService) *RecommendationsHandler {
	return &RecommendationsHandler{service: service}
}

func (h *RecommendationsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	limit := parseIntWithDefault(strings.TrimSpace(r.URL.Query().Get("limit")), 20)
	if limit <= 0 {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	items, err := h.service.Get(r.Context(), userID, limit)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}
