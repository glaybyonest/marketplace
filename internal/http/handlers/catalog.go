package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/dto"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type CatalogService interface {
	ListCategoriesTree(ctx context.Context) ([]domain.CategoryNode, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (domain.Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (domain.Category, error)
	ListProducts(ctx context.Context, filter domain.ProductFilter) (domain.PageResult[domain.Product], error)
	GetProductByID(ctx context.Context, id uuid.UUID) (domain.Product, error)
	GetProductBySlug(ctx context.Context, slug string) (domain.Product, error)
	ListReviews(ctx context.Context, productID uuid.UUID, limit int) ([]domain.Review, error)
	AddReview(ctx context.Context, userID, productID uuid.UUID, rating int, comment string) (domain.Review, error)
	SearchSuggestions(ctx context.Context, query string, limit int) ([]domain.SearchSuggestion, error)
	PopularSearches(ctx context.Context, limit int) ([]domain.PopularSearch, error)
	TrackView(ctx context.Context, userID, productID uuid.UUID) error
}

type CatalogHandler struct {
	service  CatalogService
	validate *validator.Validate
}

func NewCatalogHandler(service CatalogService) *CatalogHandler {
	return &CatalogHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *CatalogHandler) CategoriesTree(w http.ResponseWriter, r *http.Request) {
	tree, err := h.service.ListCategoriesTree(r.Context())
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, tree)
}

func (h *CatalogHandler) CategoryByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	item, err := h.service.GetCategoryByID(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *CatalogHandler) CategoryBySlug(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetCategoryBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func (h *CatalogHandler) ProductsList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := domain.ProductFilter{
		Query: strings.TrimSpace(query.Get("q")),
		Sort:  strings.TrimSpace(query.Get("sort")),
		Page:  parseIntWithDefault(query.Get("page"), 1),
		Limit: parseIntWithDefault(query.Get("limit"), 20),
	}

	if rawCategoryID := strings.TrimSpace(query.Get("category_id")); rawCategoryID != "" {
		categoryID, err := uuid.Parse(rawCategoryID)
		if err != nil {
			writeDomainError(w, r, domain.ErrInvalidInput)
			return
		}
		filter.CategoryID = &categoryID
	}
	if minPrice, ok, err := parseOptionalFloat(query.Get("min_price")); err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	} else if ok {
		filter.MinPrice = &minPrice
	}
	if maxPrice, ok, err := parseOptionalFloat(query.Get("max_price")); err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	} else if ok {
		filter.MaxPrice = &maxPrice
	}
	if inStock, ok, err := parseOptionalBool(query.Get("in_stock")); err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	} else if ok {
		filter.InStock = &inStock
	}

	result, err := h.service.ListProducts(r.Context(), filter)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *CatalogHandler) ProductByID(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	product, err := h.service.GetProductByID(r.Context(), productID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, product)
}

func (h *CatalogHandler) ProductBySlug(w http.ResponseWriter, r *http.Request) {
	product, err := h.service.GetProductBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, product)
}

func (h *CatalogHandler) ProductReviews(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	limit := parseIntWithDefault(strings.TrimSpace(r.URL.Query().Get("limit")), 20)
	items, err := h.service.ListReviews(r.Context(), productID, limit)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *CatalogHandler) ProductReviewCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	productID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	}

	var req dto.CreateProductReviewRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	item, err := h.service.AddReview(r.Context(), userID, productID, req.Rating, req.Comment)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, item)
}

func (h *CatalogHandler) SearchSuggestions(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := parseIntWithDefault(r.URL.Query().Get("limit"), 8)

	items, err := h.service.SearchSuggestions(r.Context(), query, limit)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *CatalogHandler) PopularSearches(w http.ResponseWriter, r *http.Request) {
	limit := parseIntWithDefault(r.URL.Query().Get("limit"), 6)

	items, err := h.service.PopularSearches(r.Context(), limit)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func parseIntWithDefault(value string, fallback int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseOptionalFloat(value string) (float64, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false, err
	}
	return parsed, true, nil
}

func parseOptionalBool(value string) (bool, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return false, false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false, err
	}
	return parsed, true, nil
}
