package usecase

import (
	"context"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
)

type OrdersRepository interface {
	Checkout(ctx context.Context, userID uuid.UUID, place domain.Place) (domain.Order, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page, limit int) (domain.PageResult[domain.Order], error)
	GetByIDForUser(ctx context.Context, orderID, userID uuid.UUID) (domain.Order, error)
}

type OrdersPlaceRepository interface {
	GetByIDForUser(ctx context.Context, placeID, userID uuid.UUID) (domain.Place, error)
}

type OrdersService struct {
	orders OrdersRepository
	places OrdersPlaceRepository
}

func NewOrdersService(orders OrdersRepository, places OrdersPlaceRepository) *OrdersService {
	return &OrdersService{
		orders: orders,
		places: places,
	}
}

func (s *OrdersService) Checkout(ctx context.Context, userID, placeID uuid.UUID) (domain.Order, error) {
	if userID == uuid.Nil {
		return domain.Order{}, domain.ErrUnauthorized
	}
	if placeID == uuid.Nil {
		return domain.Order{}, domain.ErrInvalidInput
	}

	place, err := s.places.GetByIDForUser(ctx, placeID, userID)
	if err != nil {
		return domain.Order{}, err
	}
	return s.orders.Checkout(ctx, userID, place)
}

func (s *OrdersService) List(ctx context.Context, userID uuid.UUID, page, limit int) (domain.PageResult[domain.Order], error) {
	if userID == uuid.Nil {
		return domain.PageResult[domain.Order]{}, domain.ErrUnauthorized
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.orders.ListByUser(ctx, userID, page, limit)
}

func (s *OrdersService) GetByID(ctx context.Context, userID, orderID uuid.UUID) (domain.Order, error) {
	if userID == uuid.Nil {
		return domain.Order{}, domain.ErrUnauthorized
	}
	if orderID == uuid.Nil {
		return domain.Order{}, domain.ErrInvalidInput
	}
	return s.orders.GetByIDForUser(ctx, orderID, userID)
}
