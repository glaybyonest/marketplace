package handlers

import (
	"context"
	"net/http"
	"strings"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/dto"
	"marketplace-backend/internal/http/response"
	"marketplace-backend/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type AdminService interface {
	ListCategories(ctx context.Context) ([]domain.Category, error)
	CreateCategory(ctx context.Context, input usecase.AdminCategoryInput) (domain.Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, input usecase.AdminCategoryInput) (domain.Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	ListProducts(ctx context.Context, filter domain.ProductFilter) (domain.PageResult[domain.Product], error)
	CreateProduct(ctx context.Context, input usecase.AdminProductInput) (domain.Product, error)
	UpdateProduct(ctx context.Context, id uuid.UUID, input usecase.AdminProductInput) (domain.Product, error)
	UpdateProductStock(ctx context.Context, id uuid.UUID, stockQty int) (domain.Product, error)
	DeleteProduct(ctx context.Context, id uuid.UUID) (domain.Product, error)
}

type AdminHandler struct {
	service  AdminService
	validate *validator.Validate
}

func NewAdminHandler(service AdminService) *AdminHandler {
	return &AdminHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *AdminHandler) CategoriesList(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListCategories(r.Context())
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *AdminHandler) CategoryCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.AdminCategoryUpsertRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	input, err := toAdminCategoryInput(req)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	category, err := h.service.CreateCategory(r.Context(), input)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, category)
}

func (h *AdminHandler) CategoryUpdate(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseUUIDParam("id", r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	var req dto.AdminCategoryUpsertRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	input, err := toAdminCategoryInput(req)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	category, err := h.service.UpdateCategory(r.Context(), categoryID, input)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, category)
}

func (h *AdminHandler) CategoryDelete(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseUUIDParam("id", r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	if err := h.service.DeleteCategory(r.Context(), categoryID); err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *AdminHandler) ProductsList(w http.ResponseWriter, r *http.Request) {
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
	if isActive, ok, err := parseOptionalBool(query.Get("is_active")); err != nil {
		writeDomainError(w, r, domain.ErrInvalidInput)
		return
	} else if ok {
		filter.IsActive = &isActive
	}

	items, err := h.service.ListProducts(r.Context(), filter)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *AdminHandler) ProductCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.AdminProductUpsertRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	input, err := toAdminProductInput(req)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	product, err := h.service.CreateProduct(r.Context(), input)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, product)
}

func (h *AdminHandler) ProductUpdate(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam("id", r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	var req dto.AdminProductUpsertRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	input, err := toAdminProductInput(req)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	product, err := h.service.UpdateProduct(r.Context(), productID, input)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, product)
}

func (h *AdminHandler) ProductUpdateStock(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam("id", r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	var req dto.AdminProductStockRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	product, err := h.service.UpdateProductStock(r.Context(), productID, req.StockQty)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, product)
}

func (h *AdminHandler) ProductDelete(w http.ResponseWriter, r *http.Request) {
	productID, err := parseUUIDParam("id", r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	product, err := h.service.DeleteProduct(r.Context(), productID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"deleted": true,
		"product": product,
	})
}

func parseUUIDParam(name string, r *http.Request) (uuid.UUID, error) {
	value := strings.TrimSpace(chi.URLParam(r, name))
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, domain.ErrInvalidInput
	}
	return parsed, nil
}

func toAdminCategoryInput(req dto.AdminCategoryUpsertRequest) (usecase.AdminCategoryInput, error) {
	var parentID *uuid.UUID
	if req.ParentID != nil {
		parsed, err := uuid.Parse(strings.TrimSpace(*req.ParentID))
		if err != nil {
			return usecase.AdminCategoryInput{}, domain.ErrInvalidInput
		}
		parentID = &parsed
	}
	return usecase.AdminCategoryInput{
		ParentID: parentID,
		Name:     req.Name,
		Slug:     req.Slug,
	}, nil
}

func toAdminProductInput(req dto.AdminProductUpsertRequest) (usecase.AdminProductInput, error) {
	categoryID, err := uuid.Parse(strings.TrimSpace(req.CategoryID))
	if err != nil {
		return usecase.AdminProductInput{}, domain.ErrInvalidInput
	}

	images := make([]string, 0, len(req.Images))
	for _, image := range req.Images {
		trimmed := strings.TrimSpace(image)
		if trimmed == "" {
			continue
		}
		images = append(images, trimmed)
	}

	return usecase.AdminProductInput{
		CategoryID:  categoryID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Price:       req.Price,
		Currency:    req.Currency,
		SKU:         req.SKU,
		ImageURL:    req.ImageURL,
		Images:      images,
		Brand:       req.Brand,
		Unit:        req.Unit,
		Specs:       req.Specs,
		StockQty:    req.StockQty,
		IsActive:    req.IsActive,
	}, nil
}
