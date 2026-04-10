package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"marketplace-backend/internal/domain"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sellerAIServiceStub struct {
	result usecase.ProductCardDraftResult
	err    error
	called bool
}

func (s *sellerAIServiceStub) GenerateProductCardDraft(ctx context.Context, sellerID uuid.UUID, input usecase.GenerateProductCardInput) (usecase.ProductCardDraftResult, error) {
	s.called = true
	if s.err != nil {
		return usecase.ProductCardDraftResult{}, s.err
	}
	return s.result, nil
}

func TestSellerAIHandler(t *testing.T) {
	sellerID := uuid.New()
	service := &sellerAIServiceStub{
		result: usecase.ProductCardDraftResult{
			Draft: usecase.GeneratedProductCardDraft{
				Name:        "Термокружка Steel Cup",
				Slug:        "termokruzhka-steel-cup",
				Description: "Аккуратный черновик карточки.",
				Brand:       "Steel Cup",
				Unit:        "шт.",
				Specs: map[string]string{
					"Объём": "450 мл",
				},
			},
			Warnings:      []string{"Проверьте характеристики вручную"},
			MissingFields: []string{"сертификаты"},
			Provider:      "openai",
			Model:         "gpt-4.1-mini",
		},
	}

	handler := NewSellerAIHandler(service)
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := r.Header.Get("X-Test-Role")
			if role == "" {
				next.ServeHTTP(w, r)
				return
			}

			ctx := httpmw.WithAuth(r.Context(), sellerID, uuid.New(), "seller@test.local", domain.UserRole(role))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.With(httpmw.RequireRole(domain.UserRoleSeller)).Post("/api/v1/seller/ai/product-card", handler.GenerateProductCard)

	t.Run("401 unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/seller/ai/product-card", strings.NewReader(`{}`))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("403 non-seller", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/seller/ai/product-card", strings.NewReader(`{}`))
		req.Header.Set("X-Test-Role", string(domain.UserRoleCustomer))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("422 invalid request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/seller/ai/product-card", strings.NewReader(`{"unknown":true}`))
		req.Header.Set("X-Test-Role", string(domain.UserRoleSeller))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("200 success", func(t *testing.T) {
		service.called = false
		req := httptest.NewRequest(http.MethodPost, "/api/v1/seller/ai/product-card", strings.NewReader(`{
			"mode":"generate",
			"category_name":"Термопосуда",
			"source_name":"Термокружка Steel Cup"
		}`))
		req.Header.Set("X-Test-Role", string(domain.UserRoleSeller))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, service.called)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "openai", data["provider"])
	})
}
