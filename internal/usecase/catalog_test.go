package usecase

import (
	"context"
	"testing"
	"time"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type categoryRepoMock struct {
	items []domain.Category
	err   error
}

func (m *categoryRepoMock) List(ctx context.Context) ([]domain.Category, error) {
	return m.items, m.err
}

func (m *categoryRepoMock) GetByID(ctx context.Context, id uuid.UUID) (domain.Category, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.Category{}, domain.ErrNotFound
}

func (m *categoryRepoMock) GetBySlug(ctx context.Context, slug string) (domain.Category, error) {
	for _, item := range m.items {
		if item.Slug == slug {
			return item, nil
		}
	}
	return domain.Category{}, domain.ErrNotFound
}

type productRepoMock struct {
	listResult  domain.PageResult[domain.Product]
	lastFilter  domain.ProductFilter
	listErr     error
	byID        map[uuid.UUID]domain.Product
	bySlug      map[string]domain.Product
	suggestions []domain.SearchSuggestion
	popular     []domain.PopularSearch
	lastTracked string
}

func (m *productRepoMock) List(ctx context.Context, filter domain.ProductFilter) (domain.PageResult[domain.Product], error) {
	m.lastFilter = filter
	return m.listResult, m.listErr
}

func (m *productRepoMock) GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	item, ok := m.byID[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	return item, nil
}

func (m *productRepoMock) GetBySlug(ctx context.Context, slug string) (domain.Product, error) {
	item, ok := m.bySlug[slug]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	return item, nil
}

func (m *productRepoMock) SearchSuggestions(ctx context.Context, query string, limit int) ([]domain.SearchSuggestion, error) {
	if len(m.suggestions) > limit {
		return m.suggestions[:limit], nil
	}
	return m.suggestions, nil
}

func (m *productRepoMock) ListPopularSearches(ctx context.Context, limit int) ([]domain.PopularSearch, error) {
	if len(m.popular) > limit {
		return m.popular[:limit], nil
	}
	return m.popular, nil
}

func (m *productRepoMock) TrackSearchQuery(ctx context.Context, query string) error {
	m.lastTracked = query
	return nil
}

type eventRepoMock struct {
	lastType string
	err      error
}

func (m *eventRepoMock) Create(ctx context.Context, userID, productID uuid.UUID, eventType string) error {
	m.lastType = eventType
	return m.err
}

func TestCatalogService(t *testing.T) {
	rootID := uuid.New()
	childID := uuid.New()
	leafID := uuid.New()
	now := time.Now().UTC()

	categories := &categoryRepoMock{
		items: []domain.Category{
			{ID: rootID, Name: "Root", Slug: "root", CreatedAt: now, UpdatedAt: now},
			{ID: childID, ParentID: &rootID, Name: "Child", Slug: "child", CreatedAt: now, UpdatedAt: now},
			{ID: leafID, ParentID: &childID, Name: "Leaf", Slug: "leaf", CreatedAt: now, UpdatedAt: now},
		},
	}

	productID := uuid.New()
	products := &productRepoMock{
		listResult: domain.PageResult[domain.Product]{Page: 1, Limit: 20, Total: 1, Items: []domain.Product{{ID: productID, Slug: "product"}}},
		byID:       map[uuid.UUID]domain.Product{productID: {ID: productID, Slug: "product"}},
		bySlug:     map[string]domain.Product{"product": {ID: productID, Slug: "product"}},
		suggestions: []domain.SearchSuggestion{
			{Text: "cement", Kind: "category"},
			{Text: "cement m500", Kind: "product"},
		},
		popular: []domain.PopularSearch{
			{Query: "cement", SearchCount: 3},
			{Query: "oak", SearchCount: 2},
		},
	}
	events := &eventRepoMock{}
	service := NewCatalogService(categories, products, events)

	t.Run("categories", func(t *testing.T) {
		tests := []struct {
			name    string
			run     func() error
			wantErr error
		}{
			{"tree success", func() error { _, err := service.ListCategoriesTree(context.Background()); return err }, nil},
			{"get category by id success", func() error { _, err := service.GetCategoryByID(context.Background(), rootID); return err }, nil},
			{"get category by id invalid", func() error { _, err := service.GetCategoryByID(context.Background(), uuid.Nil); return err }, domain.ErrInvalidInput},
			{"get category by id not found", func() error { _, err := service.GetCategoryByID(context.Background(), uuid.New()); return err }, domain.ErrNotFound},
			{"get category by slug success", func() error { _, err := service.GetCategoryBySlug(context.Background(), "root"); return err }, nil},
			{"get category by slug empty", func() error { _, err := service.GetCategoryBySlug(context.Background(), " "); return err }, domain.ErrInvalidInput},
			{"get category by slug not found", func() error { _, err := service.GetCategoryBySlug(context.Background(), "x"); return err }, domain.ErrNotFound},
			{"tree structure nested", func() error {
				tree, err := service.ListCategoriesTree(context.Background())
				if err != nil {
					return err
				}
				if len(tree) != 1 || len(tree[0].Children) != 1 || len(tree[0].Children[0].Children) != 1 {
					return domain.ErrInvalidInput
				}
				return nil
			}, nil},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.run()
				if tc.wantErr == nil {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
					assert.ErrorIs(t, err, tc.wantErr)
				}
			})
		}
	})

	t.Run("products", func(t *testing.T) {
		categoryID := uuid.New()
		tests := []struct {
			name       string
			filter     domain.ProductFilter
			wantErr    error
			assertions func(t *testing.T)
		}{
			{"list defaults", domain.ProductFilter{}, nil, func(t *testing.T) {
				assert.Equal(t, 1, products.lastFilter.Page)
				assert.Equal(t, 20, products.lastFilter.Limit)
				assert.Equal(t, domain.SortNew, products.lastFilter.Sort)
			}},
			{"list sort asc", domain.ProductFilter{Sort: domain.SortPriceAsc}, nil, nil},
			{"list sort desc", domain.ProductFilter{Sort: domain.SortPriceDesc}, nil, nil},
			{"list sort new", domain.ProductFilter{Sort: domain.SortNew}, nil, nil},
			{"list invalid sort", domain.ProductFilter{Sort: "bad"}, domain.ErrInvalidInput, nil},
			{"list page normalized", domain.ProductFilter{Page: -1, Limit: 0}, nil, func(t *testing.T) {
				assert.Equal(t, 1, products.lastFilter.Page)
				assert.Equal(t, 20, products.lastFilter.Limit)
			}},
			{"list limit max", domain.ProductFilter{Limit: 1000}, nil, func(t *testing.T) {
				assert.Equal(t, 100, products.lastFilter.Limit)
			}},
			{"list category filter", domain.ProductFilter{CategoryID: &categoryID}, nil, func(t *testing.T) {
				require.NotNil(t, products.lastFilter.CategoryID)
				assert.Equal(t, categoryID, *products.lastFilter.CategoryID)
			}},
			{"list price and stock filters", domain.ProductFilter{MinPrice: ptrFloat(10), MaxPrice: ptrFloat(50), InStock: ptrBool(true)}, nil, func(t *testing.T) {
				require.NotNil(t, products.lastFilter.MinPrice)
				require.NotNil(t, products.lastFilter.MaxPrice)
				require.NotNil(t, products.lastFilter.InStock)
				assert.Equal(t, 10.0, *products.lastFilter.MinPrice)
				assert.Equal(t, 50.0, *products.lastFilter.MaxPrice)
				assert.True(t, *products.lastFilter.InStock)
			}},
			{"list invalid price range", domain.ProductFilter{MinPrice: ptrFloat(100), MaxPrice: ptrFloat(50)}, domain.ErrInvalidInput, nil},
			{"get product by id success", domain.ProductFilter{}, nil, nil},
			{"get product by id invalid", domain.ProductFilter{}, domain.ErrInvalidInput, nil},
			{"get product by id not found", domain.ProductFilter{}, domain.ErrNotFound, nil},
			{"get product by slug success", domain.ProductFilter{}, nil, nil},
			{"get product by slug empty", domain.ProductFilter{}, domain.ErrInvalidInput, nil},
			{"get product by slug not found", domain.ProductFilter{}, domain.ErrNotFound, nil},
		}

		for index, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				var err error
				switch index {
				case 10:
					_, err = service.GetProductByID(context.Background(), productID)
				case 11:
					_, err = service.GetProductByID(context.Background(), uuid.Nil)
				case 12:
					_, err = service.GetProductByID(context.Background(), uuid.New())
				case 13:
					_, err = service.GetProductBySlug(context.Background(), "product")
				case 14:
					_, err = service.GetProductBySlug(context.Background(), " ")
				case 15:
					_, err = service.GetProductBySlug(context.Background(), "missing")
				default:
					_, err = service.ListProducts(context.Background(), tc.filter)
				}

				if tc.wantErr == nil {
					require.NoError(t, err)
					if tc.assertions != nil {
						tc.assertions(t)
					}
				} else {
					require.Error(t, err)
					assert.ErrorIs(t, err, tc.wantErr)
				}
			})
		}
	})

	t.Run("search", func(t *testing.T) {
		suggestions, err := service.SearchSuggestions(context.Background(), "  cement   m500 ", 10)
		require.NoError(t, err)
		require.Len(t, suggestions, 2)

		popular, err := service.PopularSearches(context.Background(), 10)
		require.NoError(t, err)
		require.Len(t, popular, 2)

		_, err = service.ListProducts(context.Background(), domain.ProductFilter{Query: "  Cement   M500 "})
		require.NoError(t, err)
		assert.Equal(t, "cement m500", products.lastTracked)
	})

	t.Run("events", func(t *testing.T) {
		tests := []struct {
			name    string
			userID  uuid.UUID
			itemID  uuid.UUID
			repoErr error
			wantErr error
		}{
			{"track view success", uuid.New(), uuid.New(), nil, nil},
			{"track view invalid user", uuid.Nil, uuid.New(), nil, domain.ErrInvalidInput},
			{"track view invalid product", uuid.New(), uuid.Nil, nil, domain.ErrInvalidInput},
			{"track view repo error", uuid.New(), uuid.New(), domain.ErrConflict, domain.ErrConflict},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				events.err = tc.repoErr
				err := service.TrackView(context.Background(), tc.userID, tc.itemID)
				if tc.wantErr == nil {
					require.NoError(t, err)
					assert.Equal(t, domain.ProductEventView, events.lastType)
				} else {
					require.Error(t, err)
					assert.ErrorIs(t, err, tc.wantErr)
				}
			})
		}
	})
}

func ptrFloat(value float64) *float64 {
	return &value
}

func ptrBool(value bool) *bool {
	return &value
}
