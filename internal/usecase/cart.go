package usecase

import (
	"context"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
)

type CartRepository interface {
	Get(ctx context.Context, userID uuid.UUID) (domain.Cart, error)
	GetItem(ctx context.Context, userID, productID uuid.UUID) (domain.CartItem, error)
	Add(ctx context.Context, userID, productID uuid.UUID, quantity int) error
	SetQuantity(ctx context.Context, userID, productID uuid.UUID, quantity int) (bool, error)
	Delete(ctx context.Context, userID, productID uuid.UUID) (bool, error)
	Clear(ctx context.Context, userID uuid.UUID) error
}

type CartProductRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error)
}

type CartService struct {
	cart     CartRepository
	products CartProductRepository
}

func NewCartService(cart CartRepository, products CartProductRepository) *CartService {
	return &CartService{
		cart:     cart,
		products: products,
	}
}

func (s *CartService) Get(ctx context.Context, userID uuid.UUID) (domain.Cart, error) {
	if userID == uuid.Nil {
		return domain.Cart{}, domain.ErrUnauthorized
	}
	return s.cart.Get(ctx, userID)
}

func (s *CartService) AddItem(ctx context.Context, userID, productID uuid.UUID, quantity int) error {
	if userID == uuid.Nil {
		return domain.ErrUnauthorized
	}
	if productID == uuid.Nil || quantity <= 0 {
		return domain.ErrInvalidInput
	}

	product, err := s.products.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	if !product.IsActive {
		return domain.ErrUnavailable
	}
	if product.StockQty < quantity {
		return domain.ErrStockShortage
	}

	currentQty := 0
	item, err := s.cart.GetItem(ctx, userID, productID)
	switch {
	case err == nil:
		currentQty = item.Quantity
	case err != nil && err != domain.ErrNotFound:
		return err
	}

	if currentQty+quantity > product.StockQty {
		return domain.ErrStockShortage
	}
	return s.cart.Add(ctx, userID, productID, quantity)
}

func (s *CartService) UpdateItem(ctx context.Context, userID, productID uuid.UUID, quantity int) error {
	if userID == uuid.Nil {
		return domain.ErrUnauthorized
	}
	if productID == uuid.Nil || quantity <= 0 {
		return domain.ErrInvalidInput
	}

	product, err := s.products.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	if !product.IsActive {
		return domain.ErrUnavailable
	}
	if quantity > product.StockQty {
		return domain.ErrStockShortage
	}

	updated, err := s.cart.SetQuantity(ctx, userID, productID, quantity)
	if err != nil {
		return err
	}
	if !updated {
		return domain.ErrNotFound
	}
	return nil
}

func (s *CartService) DeleteItem(ctx context.Context, userID, productID uuid.UUID) error {
	if userID == uuid.Nil {
		return domain.ErrUnauthorized
	}
	if productID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	deleted, err := s.cart.Delete(ctx, userID, productID)
	if err != nil {
		return err
	}
	if !deleted {
		return domain.ErrNotFound
	}
	return nil
}

func (s *CartService) Clear(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return domain.ErrUnauthorized
	}
	return s.cart.Clear(ctx, userID)
}
