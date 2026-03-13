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

type ordersRepoMock struct {
	lastCheckoutPlaceID uuid.UUID
	order               domain.Order
}

func (m *ordersRepoMock) Checkout(ctx context.Context, userID uuid.UUID, place domain.Place) (domain.Order, error) {
	m.lastCheckoutPlaceID = place.ID
	if m.order.ID == uuid.Nil {
		m.order = domain.Order{
			ID:          uuid.New(),
			UserID:      userID,
			PlaceID:     place.ID,
			PlaceTitle:  place.Title,
			AddressText: place.AddressText,
			Status:      domain.OrderStatusPending,
			Currency:    "RUB",
			TotalAmount: 1200,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
	}
	return m.order, nil
}

func (m *ordersRepoMock) ListByUser(ctx context.Context, userID uuid.UUID, page, limit int) (domain.PageResult[domain.Order], error) {
	return domain.PageResult[domain.Order]{
		Items: []domain.Order{m.order},
		Page:  page,
		Limit: limit,
		Total: 1,
	}, nil
}

func (m *ordersRepoMock) GetByIDForUser(ctx context.Context, orderID, userID uuid.UUID) (domain.Order, error) {
	if m.order.ID != orderID {
		return domain.Order{}, domain.ErrNotFound
	}
	return m.order, nil
}

type ordersPlaceRepoMock struct {
	places map[uuid.UUID]domain.Place
}

func (m *ordersPlaceRepoMock) GetByIDForUser(ctx context.Context, placeID, userID uuid.UUID) (domain.Place, error) {
	place, ok := m.places[placeID]
	if !ok || place.UserID != userID {
		return domain.Place{}, domain.ErrNotFound
	}
	return place, nil
}

func TestOrdersService(t *testing.T) {
	userID := uuid.New()
	placeID := uuid.New()
	placeRepo := &ordersPlaceRepoMock{
		places: map[uuid.UUID]domain.Place{
			placeID: {
				ID:          placeID,
				UserID:      userID,
				Title:       "Home",
				AddressText: "Main street",
			},
		},
	}
	orderRepo := &ordersRepoMock{}
	service := NewOrdersService(orderRepo, placeRepo)

	order, err := service.Checkout(context.Background(), userID, placeID)
	require.NoError(t, err)
	require.Equal(t, placeID, orderRepo.lastCheckoutPlaceID)
	require.Equal(t, placeID, order.PlaceID)

	tests := []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{"checkout unauthorized", func() error {
			_, err := service.Checkout(context.Background(), uuid.Nil, placeID)
			return err
		}, domain.ErrUnauthorized},
		{"checkout invalid place", func() error {
			_, err := service.Checkout(context.Background(), userID, uuid.Nil)
			return err
		}, domain.ErrInvalidInput},
		{"list unauthorized", func() error {
			_, err := service.List(context.Background(), uuid.Nil, 1, 20)
			return err
		}, domain.ErrUnauthorized},
		{"get unauthorized", func() error {
			_, err := service.GetByID(context.Background(), uuid.Nil, order.ID)
			return err
		}, domain.ErrUnauthorized},
		{"get invalid id", func() error {
			_, err := service.GetByID(context.Background(), userID, uuid.Nil)
			return err
		}, domain.ErrInvalidInput},
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
