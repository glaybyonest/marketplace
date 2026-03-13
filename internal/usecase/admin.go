package usecase

import (
	"context"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/observability"
)

type AdminCategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Category, error)
	Create(ctx context.Context, parentID *uuid.UUID, name, slug string) (domain.Category, error)
	Update(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, name, slug string) (domain.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
	HasChildren(ctx context.Context, id uuid.UUID) (bool, error)
	HasProducts(ctx context.Context, id uuid.UUID) (bool, error)
}

type AdminProductRepository interface {
	List(ctx context.Context, filter domain.ProductFilter) (domain.PageResult[domain.Product], error)
	GetByIDAny(ctx context.Context, id uuid.UUID) (domain.Product, error)
	Create(ctx context.Context, input ProductWriteInput) (domain.Product, error)
	Update(ctx context.Context, input ProductWriteInput) (domain.Product, error)
	SetActive(ctx context.Context, id uuid.UUID, isActive bool) (domain.Product, error)
	UpdateStock(ctx context.Context, id uuid.UUID, stockQty int) (domain.Product, error)
}

type AdminAuditLogger interface {
	Record(ctx context.Context, entry observability.AuditEntry) error
}

type AdminCategoryInput struct {
	ParentID *uuid.UUID
	Name     string
	Slug     string
}

type AdminProductInput struct {
	CategoryID  uuid.UUID
	Name        string
	Slug        string
	Description string
	Price       float64
	Currency    string
	SKU         string
	ImageURL    string
	Images      []string
	Brand       string
	Unit        string
	Specs       map[string]any
	StockQty    int
	IsActive    *bool
}

type ProductWriteInput struct {
	ID          uuid.UUID
	CategoryID  uuid.UUID
	Name        string
	Slug        string
	Description string
	Price       float64
	Currency    string
	SKU         string
	ImageURL    string
	Gallery     []string
	Brand       string
	Unit        string
	Specs       map[string]any
	StockQty    int
	IsActive    bool
}

type AdminService struct {
	categories AdminCategoryRepository
	products   AdminProductRepository
	audit      AdminAuditLogger
}

func NewAdminService(categories AdminCategoryRepository, products AdminProductRepository, audit AdminAuditLogger) *AdminService {
	return &AdminService{
		categories: categories,
		products:   products,
		audit:      audit,
	}
}

func (s *AdminService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	items, err := s.categories.List(ctx)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		Action:     "admin.categories_listed",
		EntityType: "category",
		Metadata: map[string]any{
			"count": len(items),
		},
	})
	return items, nil
}

func (s *AdminService) CreateCategory(ctx context.Context, input AdminCategoryInput) (domain.Category, error) {
	name, slug, err := normalizeCategoryPayload(input.Name, input.Slug)
	if err != nil {
		return domain.Category{}, err
	}
	if err := s.validateCategoryParent(ctx, uuid.Nil, input.ParentID); err != nil {
		return domain.Category{}, err
	}
	category, err := s.categories.Create(ctx, input.ParentID, name, slug)
	if err != nil {
		return domain.Category{}, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		Action:     "admin.category_created",
		EntityType: "category",
		EntityID:   ptrUUID(category.ID),
		Metadata:   categoryAuditMetadata(category),
	})
	return category, nil
}

func (s *AdminService) UpdateCategory(ctx context.Context, id uuid.UUID, input AdminCategoryInput) (domain.Category, error) {
	if id == uuid.Nil {
		return domain.Category{}, domain.ErrInvalidInput
	}
	current, err := s.categories.GetByID(ctx, id)
	if err != nil {
		return domain.Category{}, err
	}

	name, slug, err := normalizeCategoryPayload(input.Name, input.Slug)
	if err != nil {
		return domain.Category{}, err
	}
	if err := s.validateCategoryParent(ctx, id, input.ParentID); err != nil {
		return domain.Category{}, err
	}
	updated, err := s.categories.Update(ctx, id, input.ParentID, name, slug)
	if err != nil {
		return domain.Category{}, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		Action:     "admin.category_updated",
		EntityType: "category",
		EntityID:   ptrUUID(updated.ID),
		Metadata: map[string]any{
			"before": categoryAuditMetadata(current),
			"after":  categoryAuditMetadata(updated),
		},
	})
	return updated, nil
}

func (s *AdminService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return domain.ErrInvalidInput
	}
	current, err := s.categories.GetByID(ctx, id)
	if err != nil {
		return err
	}

	hasChildren, err := s.categories.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return domain.ErrConflict
	}

	hasProducts, err := s.categories.HasProducts(ctx, id)
	if err != nil {
		return err
	}
	if hasProducts {
		return domain.ErrConflict
	}

	if err := s.categories.Delete(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		Action:     "admin.category_deleted",
		EntityType: "category",
		EntityID:   ptrUUID(current.ID),
		Metadata:   categoryAuditMetadata(current),
	})
	return nil
}

func (s *AdminService) ListProducts(ctx context.Context, filter domain.ProductFilter) (domain.PageResult[domain.Product], error) {
	filter.IncludeInactive = true
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	filter.Query = strings.TrimSpace(filter.Query)
	if filter.MinPrice != nil && *filter.MinPrice < 0 {
		return domain.PageResult[domain.Product]{}, domain.ErrInvalidInput
	}
	if filter.MaxPrice != nil && *filter.MaxPrice < 0 {
		return domain.PageResult[domain.Product]{}, domain.ErrInvalidInput
	}
	if filter.MinPrice != nil && filter.MaxPrice != nil && *filter.MinPrice > *filter.MaxPrice {
		return domain.PageResult[domain.Product]{}, domain.ErrInvalidInput
	}
	filter.Sort = strings.TrimSpace(filter.Sort)
	if filter.Sort == "" {
		filter.Sort = domain.SortNew
	}
	switch filter.Sort {
	case domain.SortNew, domain.SortPriceAsc, domain.SortPriceDesc:
	default:
		return domain.PageResult[domain.Product]{}, domain.ErrInvalidInput
	}
	result, err := s.products.List(ctx, filter)
	if err != nil {
		return domain.PageResult[domain.Product]{}, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		Action:     "admin.products_listed",
		EntityType: "product",
		Metadata: map[string]any{
			"query":       filter.Query,
			"category_id": uuidToString(filter.CategoryID),
			"min_price":   floatPtrValue(filter.MinPrice),
			"max_price":   floatPtrValue(filter.MaxPrice),
			"in_stock":    boolPtrValue(filter.InStock),
			"is_active":   boolPtrValue(filter.IsActive),
			"sort":        filter.Sort,
			"page":        filter.Page,
			"limit":       filter.Limit,
			"items_count": len(result.Items),
			"total_items": result.Total,
		},
	})
	return result, nil
}

func (s *AdminService) CreateProduct(ctx context.Context, input AdminProductInput) (domain.Product, error) {
	writeInput, err := s.normalizeProductInput(ctx, domain.Product{}, input)
	if err != nil {
		return domain.Product{}, err
	}
	product, err := s.products.Create(ctx, writeInput)
	if err != nil {
		return domain.Product{}, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		Action:     "admin.product_created",
		EntityType: "product",
		EntityID:   ptrUUID(product.ID),
		Metadata:   productAuditMetadata(product),
	})
	return product, nil
}

func (s *AdminService) UpdateProduct(ctx context.Context, id uuid.UUID, input AdminProductInput) (domain.Product, error) {
	current, err := s.products.GetByIDAny(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}

	writeInput, err := s.normalizeProductInput(ctx, current, input)
	if err != nil {
		return domain.Product{}, err
	}
	updated, err := s.products.Update(ctx, writeInput)
	if err != nil {
		return domain.Product{}, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		Action:     "admin.product_updated",
		EntityType: "product",
		EntityID:   ptrUUID(updated.ID),
		Metadata: map[string]any{
			"before": productAuditMetadata(current),
			"after":  productAuditMetadata(updated),
		},
	})
	return updated, nil
}

func (s *AdminService) UpdateProductStock(ctx context.Context, id uuid.UUID, stockQty int) (domain.Product, error) {
	if id == uuid.Nil || stockQty < 0 {
		return domain.Product{}, domain.ErrInvalidInput
	}
	current, err := s.products.GetByIDAny(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}
	updated, err := s.products.UpdateStock(ctx, id, stockQty)
	if err != nil {
		return domain.Product{}, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		Action:     "admin.product_stock_updated",
		EntityType: "product",
		EntityID:   ptrUUID(updated.ID),
		Metadata: map[string]any{
			"sku":              updated.SKU,
			"before_stock_qty": current.StockQty,
			"after_stock_qty":  updated.StockQty,
		},
	})
	return updated, nil
}

func (s *AdminService) DeleteProduct(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	if id == uuid.Nil {
		return domain.Product{}, domain.ErrInvalidInput
	}
	current, err := s.products.GetByIDAny(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}
	updated, err := s.products.SetActive(ctx, id, false)
	if err != nil {
		return domain.Product{}, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		Action:     "admin.product_deleted",
		EntityType: "product",
		EntityID:   ptrUUID(updated.ID),
		Metadata: map[string]any{
			"soft_delete": true,
			"before":      productAuditMetadata(current),
			"after":       productAuditMetadata(updated),
		},
	})
	return updated, nil
}

func (s *AdminService) validateCategoryParent(ctx context.Context, categoryID uuid.UUID, parentID *uuid.UUID) error {
	if parentID == nil {
		return nil
	}
	if *parentID == uuid.Nil {
		return domain.ErrInvalidInput
	}
	if categoryID != uuid.Nil && *parentID == categoryID {
		return domain.ErrInvalidInput
	}
	if _, err := s.categories.GetByID(ctx, *parentID); err != nil {
		return err
	}
	if categoryID == uuid.Nil {
		return nil
	}

	items, err := s.categories.List(ctx)
	if err != nil {
		return err
	}

	parentByID := make(map[uuid.UUID]*uuid.UUID, len(items))
	for _, item := range items {
		parentByID[item.ID] = item.ParentID
	}

	current := parentID
	for current != nil {
		if *current == categoryID {
			return domain.ErrInvalidInput
		}
		current = parentByID[*current]
	}
	return nil
}

func (s *AdminService) normalizeProductInput(ctx context.Context, current domain.Product, input AdminProductInput) (ProductWriteInput, error) {
	if input.CategoryID == uuid.Nil {
		return ProductWriteInput{}, domain.ErrInvalidInput
	}
	if _, err := s.categories.GetByID(ctx, input.CategoryID); err != nil {
		return ProductWriteInput{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 180 {
		return ProductWriteInput{}, domain.ErrInvalidInput
	}

	slug := normalizeSlug(input.Slug)
	if slug == "" {
		slug = normalizeSlug(name)
	}
	if slug == "" {
		return ProductWriteInput{}, domain.ErrInvalidInput
	}

	sku := strings.TrimSpace(input.SKU)
	if sku == "" || len(sku) > 120 {
		return ProductWriteInput{}, domain.ErrInvalidInput
	}

	if input.Price < 0 || input.StockQty < 0 {
		return ProductWriteInput{}, domain.ErrInvalidInput
	}

	currency := normalizeCurrency(input.Currency)
	if currency == "" {
		return ProductWriteInput{}, domain.ErrInvalidInput
	}

	specs, err := normalizeSpecs(input.Specs)
	if err != nil {
		return ProductWriteInput{}, err
	}

	isActive := true
	if current.ID != uuid.Nil {
		isActive = current.IsActive
	}
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	return ProductWriteInput{
		ID:          current.ID,
		CategoryID:  input.CategoryID,
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(input.Description),
		Price:       input.Price,
		Currency:    currency,
		SKU:         sku,
		ImageURL:    strings.TrimSpace(input.ImageURL),
		Gallery:     normalizeStringList(input.Images),
		Brand:       strings.TrimSpace(input.Brand),
		Unit:        strings.TrimSpace(input.Unit),
		Specs:       specs,
		StockQty:    input.StockQty,
		IsActive:    isActive,
	}, nil
}

func (s *AdminService) recordAudit(ctx context.Context, entry observability.AuditEntry) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, entry)
}

func categoryAuditMetadata(category domain.Category) map[string]any {
	return map[string]any{
		"name":      category.Name,
		"slug":      category.Slug,
		"parent_id": uuidToString(category.ParentID),
	}
}

func productAuditMetadata(product domain.Product) map[string]any {
	return map[string]any{
		"category_id":   product.CategoryID.String(),
		"category_name": product.CategoryName,
		"name":          product.Name,
		"slug":          product.Slug,
		"sku":           product.SKU,
		"price":         product.Price,
		"currency":      product.Currency,
		"stock_qty":     product.StockQty,
		"is_active":     product.IsActive,
		"brand":         product.Brand,
		"unit":          product.Unit,
	}
}

func uuidToString(value *uuid.UUID) any {
	if value == nil || *value == uuid.Nil {
		return nil
	}
	return value.String()
}

func floatPtrValue(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func boolPtrValue(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func normalizeCategoryPayload(name, slug string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return "", "", domain.ErrInvalidInput
	}

	slug = normalizeSlug(slug)
	if slug == "" {
		slug = normalizeSlug(name)
	}
	if slug == "" {
		return "", "", domain.ErrInvalidInput
	}

	return name, slug, nil
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_':
			if builder.Len() == 0 || lastDash {
				continue
			}
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func normalizeCurrency(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "RUB"
	}
	if len(value) != 3 {
		return ""
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return value
}

func normalizeStringList(items []string) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func normalizeSpecs(input map[string]any) (map[string]any, error) {
	if input == nil {
		return map[string]any{}, nil
	}

	result := make(map[string]any, len(input))
	for key, value := range input {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		switch typed := value.(type) {
		case nil:
			result[key] = nil
		case string:
			result[key] = strings.TrimSpace(typed)
		case bool:
			result[key] = typed
		case float64:
			result[key] = typed
		case float32:
			result[key] = float64(typed)
		case int:
			result[key] = float64(typed)
		case int8:
			result[key] = float64(typed)
		case int16:
			result[key] = float64(typed)
		case int32:
			result[key] = float64(typed)
		case int64:
			result[key] = float64(typed)
		case uint:
			result[key] = float64(typed)
		case uint8:
			result[key] = float64(typed)
		case uint16:
			result[key] = float64(typed)
		case uint32:
			result[key] = float64(typed)
		case uint64:
			result[key] = float64(typed)
		default:
			return nil, domain.ErrInvalidInput
		}
	}
	return result, nil
}
