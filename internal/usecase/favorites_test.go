package usecase

import (
	"context"
	"testing"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type favoritesRepoMock struct {
	items       map[uuid.UUID]map[uuid.UUID]struct{}
	productList map[uuid.UUID]domain.Product
	addErr      error
	removeErr   error
	listErr     error
}

func (m *favoritesRepoMock) Add(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	if m.addErr != nil {
		return false, m.addErr
	}
	if _, ok := m.items[userID]; !ok {
		m.items[userID] = map[uuid.UUID]struct{}{}
	}
	_, existed := m.items[userID][productID]
	m.items[userID][productID] = struct{}{}
	return !existed, nil
}

func (m *favoritesRepoMock) Remove(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	if m.removeErr != nil {
		return false, m.removeErr
	}
	if _, ok := m.items[userID]; !ok {
		return false, nil
	}
	_, existed := m.items[userID][productID]
	delete(m.items[userID], productID)
	return existed, nil
}

func (m *favoritesRepoMock) List(ctx context.Context, userID uuid.UUID, page, limit int) (domain.PageResult[domain.Product], error) {
	if m.listErr != nil {
		return domain.PageResult[domain.Product]{}, m.listErr
	}
	items := make([]domain.Product, 0, limit)
	for productID := range m.items[userID] {
		items = append(items, m.productList[productID])
		if len(items) == limit {
			break
		}
	}
	return domain.PageResult[domain.Product]{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: len(m.items[userID]),
	}, nil
}

type favoritesProductRepoMock struct {
	products map[uuid.UUID]domain.Product
}

func (m *favoritesProductRepoMock) GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	item, ok := m.products[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	return item, nil
}

type favoritesEventRepoMock struct {
	calls int
	err   error
}

func (m *favoritesEventRepoMock) Create(ctx context.Context, userID, productID uuid.UUID, eventType string) error {
	m.calls++
	return m.err
}

func TestFavoritesService(t *testing.T) {
	userID := uuid.New()
	productID := uuid.New()
	secondProductID := uuid.New()

	repo := &favoritesRepoMock{
		items: map[uuid.UUID]map[uuid.UUID]struct{}{},
		productList: map[uuid.UUID]domain.Product{
			productID:       {ID: productID, Slug: "one"},
			secondProductID: {ID: secondProductID, Slug: "two"},
		},
	}
	productRepo := &favoritesProductRepoMock{
		products: map[uuid.UUID]domain.Product{
			productID:       {ID: productID, Slug: "one"},
			secondProductID: {ID: secondProductID, Slug: "two"},
		},
	}
	eventRepo := &favoritesEventRepoMock{}
	service := NewFavoritesService(repo, productRepo, eventRepo)

	tests := []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{"add success", func() error { _, err := service.Add(context.Background(), userID, productID); return err }, nil},
		{"add duplicate idempotent", func() error { _, err := service.Add(context.Background(), userID, productID); return err }, nil},
		{"add second product", func() error { _, err := service.Add(context.Background(), userID, secondProductID); return err }, nil},
		{"add unauthorized user", func() error { _, err := service.Add(context.Background(), uuid.Nil, productID); return err }, domain.ErrInvalidInput},
		{"add invalid product id", func() error { _, err := service.Add(context.Background(), userID, uuid.Nil); return err }, domain.ErrInvalidInput},
		{"add unknown product", func() error { _, err := service.Add(context.Background(), userID, uuid.New()); return err }, domain.ErrNotFound},
		{"list success", func() error { _, err := service.List(context.Background(), userID, 1, 20); return err }, nil},
		{"list default page", func() error { _, err := service.List(context.Background(), userID, 0, 0); return err }, nil},
		{"list max limit", func() error {
			result, err := service.List(context.Background(), userID, 1, 1000)
			if err == nil && result.Limit != 100 {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"list unauthorized", func() error { _, err := service.List(context.Background(), uuid.Nil, 1, 20); return err }, domain.ErrUnauthorized},
		{"remove success", func() error { return service.Remove(context.Background(), userID, productID) }, nil},
		{"remove idempotent", func() error { return service.Remove(context.Background(), userID, productID) }, nil},
		{"remove unknown user", func() error { return service.Remove(context.Background(), uuid.New(), productID) }, nil},
		{"remove invalid user", func() error { return service.Remove(context.Background(), uuid.Nil, productID) }, domain.ErrInvalidInput},
		{"remove invalid product", func() error { return service.Remove(context.Background(), userID, uuid.Nil) }, domain.ErrInvalidInput},
		{"add event called once", func() error {
			before := eventRepo.calls
			_, err := service.Add(context.Background(), userID, productID)
			if err == nil && eventRepo.calls-before != 1 {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"add duplicate event not called", func() error {
			before := eventRepo.calls
			_, err := service.Add(context.Background(), userID, productID)
			if err == nil && eventRepo.calls != before {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"event failure ignored", func() error {
			eventRepo.err = domain.ErrConflict
			defer func() { eventRepo.err = nil }()
			_, err := service.Add(context.Background(), userID, secondProductID)
			return err
		}, nil},
		{"repo add error", func() error {
			repo.addErr = domain.ErrConflict
			defer func() { repo.addErr = nil }()
			_, err := service.Add(context.Background(), userID, secondProductID)
			return err
		}, domain.ErrConflict},
		{"repo remove error", func() error {
			repo.removeErr = domain.ErrConflict
			defer func() { repo.removeErr = nil }()
			return service.Remove(context.Background(), userID, secondProductID)
		}, domain.ErrConflict},
		{"repo list error", func() error {
			repo.listErr = domain.ErrConflict
			defer func() { repo.listErr = nil }()
			_, err := service.List(context.Background(), userID, 1, 20)
			return err
		}, domain.ErrConflict},
		{"list page negative normalized", func() error {
			result, err := service.List(context.Background(), userID, -3, -10)
			if err == nil && (result.Page != 1 || result.Limit != 20) {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"list zero items", func() error {
			other := uuid.New()
			result, err := service.List(context.Background(), other, 1, 20)
			if err == nil && result.Total != 0 {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"add after remove", func() error {
			_ = service.Remove(context.Background(), userID, productID)
			_, err := service.Add(context.Background(), userID, productID)
			return err
		}, nil},
		{"remove after list", func() error {
			_, _ = service.List(context.Background(), userID, 1, 20)
			return service.Remove(context.Background(), userID, secondProductID)
		}, nil},
		{"add many items", func() error {
			for i := 0; i < 5; i++ {
				id := uuid.New()
				productRepo.products[id] = domain.Product{ID: id}
				repo.productList[id] = domain.Product{ID: id}
				_, err := service.Add(context.Background(), userID, id)
				if err != nil {
					return err
				}
			}
			return nil
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
