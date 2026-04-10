package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/dto"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/http/response"
	"marketplace-backend/internal/usecase"

	"github.com/google/uuid"
)

const sellerAIRequestBodyLimit = 64 << 10

type SellerAIService interface {
	GenerateProductCardDraft(ctx context.Context, sellerID uuid.UUID, input usecase.GenerateProductCardInput) (usecase.ProductCardDraftResult, error)
}

type SellerAIHandler struct {
	service SellerAIService
}

func NewSellerAIHandler(service SellerAIService) *SellerAIHandler {
	return &SellerAIHandler{service: service}
}

func (h *SellerAIHandler) GenerateProductCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	req, err := decodeSellerAIProductCardRequest(w, r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	existingDraft := usecase.GeneratedProductCardDraft{}
	if req.ExistingDraft != nil {
		existingDraft = usecase.GeneratedProductCardDraft{
			Name:        req.ExistingDraft.Name,
			Slug:        req.ExistingDraft.Slug,
			Description: req.ExistingDraft.Description,
			Brand:       req.ExistingDraft.Brand,
			Unit:        req.ExistingDraft.Unit,
			Specs:       req.ExistingDraft.Specs,
		}
	}

	result, err := h.service.GenerateProductCardDraft(r.Context(), userID, usecase.GenerateProductCardInput{
		Mode:           req.Mode,
		CategoryID:     req.CategoryID,
		CategoryName:   req.CategoryName,
		SourceName:     req.SourceName,
		RawDescription: req.RawDescription,
		Features:       req.Features,
		Brand:          req.Brand,
		Unit:           req.Unit,
		Specs:          req.Specs,
		Keywords:       req.Keywords,
		Tone:           req.Tone,
		ExistingDraft:  existingDraft,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func decodeSellerAIProductCardRequest(w http.ResponseWriter, r *http.Request) (dto.SellerAIProductCardRequest, error) {
	if r == nil || r.Body == nil {
		return dto.SellerAIProductCardRequest{}, domain.ErrUnprocessable
	}

	r.Body = http.MaxBytesReader(w, r.Body, sellerAIRequestBodyLimit)

	var req dto.SellerAIProductCardRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return dto.SellerAIProductCardRequest{}, domain.ErrUnprocessable
		}
		return dto.SellerAIProductCardRequest{}, domain.ErrUnprocessable
	}
	if decoder.More() {
		return dto.SellerAIProductCardRequest{}, domain.ErrUnprocessable
	}

	return req, nil
}
