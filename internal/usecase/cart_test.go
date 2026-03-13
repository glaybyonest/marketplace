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

type cartRepoMock struct {
	items map[uuid.UUID]map[uuid.UUID]int
}

func (m *cartRepoMock) Get(ctx context.Context, userID uuid.UUID) (domain.Cart, error) {
	userItems := m.items[userID]
	result := domain.Cart{
		Items:      make([]domain.CartItem, 0, len(userItems)),
		Currency:   "RUB",
		TotalItems: 0,
	}

	for productID, quantity := range userItems {
		item := domain.CartItem{
			ID:        productID,
			ProductID: productID,
			Name:      "Item",
			Price:     100,
			Quantity:  quantity,
			LineTotal: float64(quantity) * 100,
			Currency:  "RUB",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		result.Items = append(result.Items, item)
		result.TotalAmount += item.LineTotal
		result.TotalItems += quantity
	}

	return result, nil
}

func (m *cartRepoMock) GetItem(ctx context.Context, userID, productID uuid.UUID) (domain.CartItem, error) {
	if m.items[userID] == nil {
		return domain.CartItem{}, domain.ErrNotFound
	}
	quantity, ok := m.items[userID][productID]
	if !ok {
		return domain.CartItem{}, domain.ErrNotFound
	}
	return domain.CartItem{
		ID:        productID,
		ProductID: productID,
		Quantity:  quantity,
		Currency:  "RUB",
	}, nil
}

func (m *cartRepoMock) Add(ctx context.Context, userID, productID uuid.UUID, quantity int) error {
	if m.items[userID] == nil {
		m.items[userID] = map[uuid.UUID]int{}
	}
	m.items[userID][productID] += quantity
	return nil
}

func (m *cartRepoMock) SetQuantity(ctx context.Context, userID, productID uuid.UUID, quantity int) (bool, error) {
	if m.items[userID] == nil {
		return false, nil
	}
	if _, ok := m.items[userID][productID]; !ok {
		return false, nil
	}
	m.items[userID][productID] = quantity
	return true, nil
}

func (m *cartRepoMock) Delete(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	if m.items[userID] == nil {
		return false, nil
	}
	if _, ok := m.items[userID][productID]; !ok {
		return false, nil
	}
	delete(m.items[userID], productID)
	return true, nil
}

func (m *cartRepoMock) Clear(ctx context.Context, userID uuid.UUID) error {
	delete(m.items, userID)
	return nil
}

type cartProductRepoMock struct {
	products map[uuid.UUID]domain.Product
}

func (m *cartProductRepoMock) GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	product, ok := m.products[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	return product, nil
}

func TestCartService(t *testing.T) {
	userID := uuid.New()
	productID := uuid.New()

	cartRepo := &cartRepoMock{items: map[uuid.UUID]map[uuid.UUID]int{}}
	productRepo := &cartProductRepoMock{
		products: map[uuid.UUID]domain.Product{
			productID: {
				ID:       productID,
				Name:     "Product",
				StockQty: 5,
				IsActive: true,
			},
		},
	}

	service := NewCartService(cartRepo, productRepo)

	tests := []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{"get unauthorized", func() error {
			_, err := service.Get(context.Background(), uuid.Nil)
			return err
		}, domain.ErrUnauthorized},
		{"add success", func() error {
			return service.AddItem(context.Background(), userID, productID, 2)
		}, nil},
		{"add exceeds stock", func() error {
			return service.AddItem(context.Background(), userID, productID, 4)
		}, domain.ErrStockShortage},
		{"update success", func() error {
			return service.UpdateItem(context.Background(), userID, productID, 3)
		}, nil},
		{"update too much", func() error {
			return service.UpdateItem(context.Background(), userID, productID, 6)
		}, domain.ErrStockShortage},
		{"delete success", func() error {
			return service.DeleteItem(context.Background(), userID, productID)
		}, nil},
		{"delete missing", func() error {
			return service.DeleteItem(context.Background(), userID, productID)
		}, domain.ErrNotFound},
		{"clear success", func() error {
			_ = service.AddItem(context.Background(), userID, productID, 1)
			return service.Clear(context.Background(), userID)
		}, nil},
	}

	for _, tc := range tests {
		tc := tc
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
}
