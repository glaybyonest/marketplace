package usecase

import (
	"context"
	"testing"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/observability"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type adminCategoryRepoMock struct {
	items       map[uuid.UUID]domain.Category
	hasChildren map[uuid.UUID]bool
	hasProducts map[uuid.UUID]bool
}

func (m *adminCategoryRepoMock) List(ctx context.Context) ([]domain.Category, error) {
	result := make([]domain.Category, 0, len(m.items))
	for _, item := range m.items {
		result = append(result, item)
	}
	return result, nil
}

func (m *adminCategoryRepoMock) GetByID(ctx context.Context, id uuid.UUID) (domain.Category, error) {
	item, ok := m.items[id]
	if !ok {
		return domain.Category{}, domain.ErrNotFound
	}
	return item, nil
}

func (m *adminCategoryRepoMock) Create(ctx context.Context, parentID *uuid.UUID, name, slug string) (domain.Category, error) {
	category := domain.Category{
		ID:        uuid.New(),
		ParentID:  parentID,
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	m.items[category.ID] = category
	return category, nil
}

func (m *adminCategoryRepoMock) Update(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, name, slug string) (domain.Category, error) {
	category, ok := m.items[id]
	if !ok {
		return domain.Category{}, domain.ErrNotFound
	}
	category.ParentID = parentID
	category.Name = name
	category.Slug = slug
	category.UpdatedAt = time.Now().UTC()
	m.items[id] = category
	return category, nil
}

func (m *adminCategoryRepoMock) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.items[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *adminCategoryRepoMock) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.hasChildren[id], nil
}

func (m *adminCategoryRepoMock) HasProducts(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.hasProducts[id], nil
}

type adminProductRepoMock struct {
	listResult      domain.PageResult[domain.Product]
	lastFilter      domain.ProductFilter
	items           map[uuid.UUID]domain.Product
	lastCreateInput ProductWriteInput
	lastUpdateInput ProductWriteInput
	lastStockID     uuid.UUID
	lastStockQty    int
	lastActiveID    uuid.UUID
	lastActiveValue bool
}

func (m *adminProductRepoMock) List(ctx context.Context, filter domain.ProductFilter) (domain.PageResult[domain.Product], error) {
	m.lastFilter = filter
	return m.listResult, nil
}

func (m *adminProductRepoMock) GetByIDAny(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	item, ok := m.items[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	return item, nil
}

func (m *adminProductRepoMock) Create(ctx context.Context, input ProductWriteInput) (domain.Product, error) {
	m.lastCreateInput = input
	product := domain.Product{
		ID:           uuid.New(),
		CategoryID:   input.CategoryID,
		Name:         input.Name,
		Slug:         input.Slug,
		Description:  input.Description,
		Price:        input.Price,
		Currency:     input.Currency,
		SKU:          input.SKU,
		ImageURL:     input.ImageURL,
		Images:       append([]string{}, input.Gallery...),
		Brand:        input.Brand,
		Unit:         input.Unit,
		Specs:        input.Specs,
		StockQty:     input.StockQty,
		IsActive:     input.IsActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		CategoryName: "Category",
	}
	m.items[product.ID] = product
	return product, nil
}

func (m *adminProductRepoMock) Update(ctx context.Context, input ProductWriteInput) (domain.Product, error) {
	m.lastUpdateInput = input
	product, ok := m.items[input.ID]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	product.CategoryID = input.CategoryID
	product.Name = input.Name
	product.Slug = input.Slug
	product.Description = input.Description
	product.Price = input.Price
	product.Currency = input.Currency
	product.SKU = input.SKU
	product.ImageURL = input.ImageURL
	product.Images = append([]string{}, input.Gallery...)
	product.Brand = input.Brand
	product.Unit = input.Unit
	product.Specs = input.Specs
	product.StockQty = input.StockQty
	product.IsActive = input.IsActive
	product.UpdatedAt = time.Now().UTC()
	m.items[input.ID] = product
	return product, nil
}

func (m *adminProductRepoMock) SetActive(ctx context.Context, id uuid.UUID, isActive bool) (domain.Product, error) {
	m.lastActiveID = id
	m.lastActiveValue = isActive
	product, ok := m.items[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	product.IsActive = isActive
	m.items[id] = product
	return product, nil
}

func (m *adminProductRepoMock) UpdateStock(ctx context.Context, id uuid.UUID, stockQty int) (domain.Product, error) {
	m.lastStockID = id
	m.lastStockQty = stockQty
	product, ok := m.items[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	product.StockQty = stockQty
	m.items[id] = product
	return product, nil
}

type adminAuditMock struct {
	entries []observability.AuditEntry
}

func (m *adminAuditMock) Record(ctx context.Context, entry observability.AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func TestAdminService(t *testing.T) {
	rootID := uuid.New()
	childID := uuid.New()
	adminID := uuid.New()
	now := time.Now().UTC()

	categories := &adminCategoryRepoMock{
		items: map[uuid.UUID]domain.Category{
			rootID:  {ID: rootID, Name: "Root", Slug: "root", CreatedAt: now, UpdatedAt: now},
			childID: {ID: childID, ParentID: &rootID, Name: "Child", Slug: "child", CreatedAt: now, UpdatedAt: now},
		},
		hasChildren: map[uuid.UUID]bool{},
		hasProducts: map[uuid.UUID]bool{},
	}

	productID := uuid.New()
	products := &adminProductRepoMock{
		listResult: domain.PageResult[domain.Product]{Items: []domain.Product{{ID: productID}}, Page: 1, Limit: 20, Total: 1},
		items: map[uuid.UUID]domain.Product{
			productID: {
				ID:         productID,
				CategoryID: rootID,
				Name:       "Existing Product",
				Slug:       "existing-product",
				SKU:        "SKU-1",
				Currency:   "RUB",
				StockQty:   4,
				IsActive:   true,
			},
		},
	}

	audit := &adminAuditMock{}
	service := NewAdminService(categories, products, audit)
	ctx := observability.WithActor(context.Background(), observability.Actor{
		UserID: adminID,
		Email:  "admin@example.com",
		Role:   string(domain.UserRoleAdmin),
	})

	t.Run("create category normalizes slug", func(t *testing.T) {
		category, err := service.CreateCategory(ctx, AdminCategoryInput{
			Name: "Fresh Category",
		})
		require.NoError(t, err)
		assert.Equal(t, "fresh-category", category.Slug)
		require.NotEmpty(t, audit.entries)
		entry := audit.entries[len(audit.entries)-1]
		assert.Equal(t, "admin.category_created", entry.Action)
		assert.Equal(t, "category", entry.EntityType)
		require.NotNil(t, entry.EntityID)
		assert.Equal(t, category.ID, *entry.EntityID)
	})

	t.Run("update category rejects cycle", func(t *testing.T) {
		_, err := service.UpdateCategory(ctx, rootID, AdminCategoryInput{
			ParentID: &childID,
			Name:     "Root",
			Slug:     "root",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("delete category rejects dependencies", func(t *testing.T) {
		categories.hasProducts[rootID] = true
		err := service.DeleteCategory(ctx, rootID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrConflict)
		categories.hasProducts[rootID] = false
	})

	t.Run("delete category is audited", func(t *testing.T) {
		category, err := service.CreateCategory(ctx, AdminCategoryInput{
			Name: "Disposable Category",
		})
		require.NoError(t, err)

		err = service.DeleteCategory(ctx, category.ID)
		require.NoError(t, err)

		entry := audit.entries[len(audit.entries)-1]
		assert.Equal(t, "admin.category_deleted", entry.Action)
		assert.Equal(t, "category", entry.EntityType)
		require.NotNil(t, entry.EntityID)
		assert.Equal(t, category.ID, *entry.EntityID)
	})

	t.Run("list categories is audited", func(t *testing.T) {
		items, err := service.ListCategories(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, items)

		entry := audit.entries[len(audit.entries)-1]
		assert.Equal(t, "admin.categories_listed", entry.Action)
		assert.Equal(t, "category", entry.EntityType)
		assert.Equal(t, len(items), entry.Metadata["count"])
	})

	t.Run("list products includes inactive", func(t *testing.T) {
		filter := domain.ProductFilter{
			Page:       1,
			Limit:      10,
			IsActive:   ptrBool(true),
			Query:      "cement",
			MinPrice:   ptrFloat(100),
			MaxPrice:   ptrFloat(500),
			InStock:    ptrBool(true),
			CategoryID: &rootID,
		}
		result, err := service.ListProducts(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		assert.True(t, products.lastFilter.IncludeInactive)
		require.NotNil(t, products.lastFilter.IsActive)
		assert.True(t, *products.lastFilter.IsActive)
		entry := audit.entries[len(audit.entries)-1]
		assert.Equal(t, "admin.products_listed", entry.Action)
		assert.Equal(t, "product", entry.EntityType)
		assert.Equal(t, "cement", entry.Metadata["query"])
		assert.Equal(t, rootID.String(), entry.Metadata["category_id"])
		assert.Equal(t, 1, entry.Metadata["items_count"])
		assert.Equal(t, 1, entry.Metadata["total_items"])
	})

	t.Run("create product normalizes fields", func(t *testing.T) {
		product, err := service.CreateProduct(ctx, AdminProductInput{
			CategoryID: rootID,
			Name:       " New Product ",
			Price:      1250.5,
			Currency:   "rub",
			SKU:        " SKU-NEW ",
			Images:     []string{"https://img.example/1.png", "https://img.example/1.png", " "},
			Specs:      map[string]any{"weight": "25kg", "stock": 12},
			StockQty:   7,
		})
		require.NoError(t, err)
		assert.Equal(t, "new-product", products.lastCreateInput.Slug)
		assert.Equal(t, "RUB", products.lastCreateInput.Currency)
		assert.True(t, products.lastCreateInput.IsActive)
		require.Len(t, products.lastCreateInput.Gallery, 1)
		assert.Equal(t, 7, product.StockQty)
		entry := audit.entries[len(audit.entries)-1]
		assert.Equal(t, "admin.product_created", entry.Action)
	})

	t.Run("update product preserves active when omitted", func(t *testing.T) {
		product, err := service.UpdateProduct(ctx, productID, AdminProductInput{
			CategoryID:  rootID,
			Name:        "Updated Product",
			Slug:        "updated-product",
			Description: "Description",
			Price:       999,
			Currency:    "USD",
			SKU:         "SKU-2",
			StockQty:    2,
		})
		require.NoError(t, err)
		assert.True(t, products.lastUpdateInput.IsActive)
		assert.Equal(t, "updated-product", product.Slug)
		entry := audit.entries[len(audit.entries)-1]
		assert.Equal(t, "admin.product_updated", entry.Action)
	})

	t.Run("update stock validates negative values", func(t *testing.T) {
		_, err := service.UpdateProductStock(ctx, productID, -1)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("update stock is audited", func(t *testing.T) {
		product, err := service.UpdateProductStock(ctx, productID, 11)
		require.NoError(t, err)
		assert.Equal(t, 11, product.StockQty)
		entry := audit.entries[len(audit.entries)-1]
		assert.Equal(t, "admin.product_stock_updated", entry.Action)
		assert.Equal(t, 11, entry.Metadata["after_stock_qty"])
	})

	t.Run("delete product is soft deactivate", func(t *testing.T) {
		product, err := service.DeleteProduct(ctx, productID)
		require.NoError(t, err)
		assert.Equal(t, productID, products.lastActiveID)
		assert.False(t, products.lastActiveValue)
		assert.False(t, product.IsActive)
		entry := audit.entries[len(audit.entries)-1]
		assert.Equal(t, "admin.product_deleted", entry.Action)
		assert.Equal(t, true, entry.Metadata["soft_delete"])
	})
}
