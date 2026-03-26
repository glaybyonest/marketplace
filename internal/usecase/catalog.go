package usecase

import (
	"context"
	"strings"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
)

type CatalogCategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Category, error)
	GetBySlug(ctx context.Context, slug string) (domain.Category, error)
}

type CatalogProductRepository interface {
	List(ctx context.Context, filter domain.ProductFilter) (domain.PageResult[domain.Product], error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Product, error)
	GetBySlug(ctx context.Context, slug string) (domain.Product, error)
	SearchSuggestions(ctx context.Context, query string, limit int) ([]domain.SearchSuggestion, error)
	ListPopularSearches(ctx context.Context, limit int) ([]domain.PopularSearch, error)
	TrackSearchQuery(ctx context.Context, query string) error
}

type CatalogReviewRepository interface {
	ListByProductID(ctx context.Context, productID uuid.UUID, limit int) ([]domain.Review, error)
	Create(ctx context.Context, productID, userID uuid.UUID, rating int, comment string) (domain.Review, error)
}

type CatalogEventRepository interface {
	Create(ctx context.Context, userID, productID uuid.UUID, eventType string) error
}

type CatalogService struct {
	categories CatalogCategoryRepository
	products   CatalogProductRepository
	events     CatalogEventRepository
	reviews    CatalogReviewRepository
}

func NewCatalogService(
	categories CatalogCategoryRepository,
	products CatalogProductRepository,
	events CatalogEventRepository,
	reviews CatalogReviewRepository,
) *CatalogService {
	return &CatalogService{
		categories: categories,
		products:   products,
		events:     events,
		reviews:    reviews,
	}
}

func (s *CatalogService) ListCategoriesTree(ctx context.Context) ([]domain.CategoryNode, error) {
	categories, err := s.categories.List(ctx)
	if err != nil {
		return nil, err
	}

	childrenByParent := make(map[uuid.UUID][]domain.Category)
	roots := make([]domain.Category, 0)
	for _, category := range categories {
		if category.ParentID == nil {
			roots = append(roots, category)
			continue
		}
		childrenByParent[*category.ParentID] = append(childrenByParent[*category.ParentID], category)
	}

	var build func(category domain.Category) domain.CategoryNode
	build = func(category domain.Category) domain.CategoryNode {
		node := domain.CategoryNode{
			ID:        category.ID,
			ParentID:  category.ParentID,
			Name:      category.Name,
			Slug:      category.Slug,
			CreatedAt: category.CreatedAt,
			UpdatedAt: category.UpdatedAt,
			Children:  make([]domain.CategoryNode, 0),
		}

		children := childrenByParent[category.ID]
		for _, child := range children {
			node.Children = append(node.Children, build(child))
		}
		return node
	}

	tree := make([]domain.CategoryNode, 0, len(roots))
	for _, root := range roots {
		tree = append(tree, build(root))
	}
	return tree, nil
}

func (s *CatalogService) GetCategoryByID(ctx context.Context, id uuid.UUID) (domain.Category, error) {
	if id == uuid.Nil {
		return domain.Category{}, domain.ErrInvalidInput
	}
	return s.categories.GetByID(ctx, id)
}

func (s *CatalogService) GetCategoryBySlug(ctx context.Context, slug string) (domain.Category, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return domain.Category{}, domain.ErrInvalidInput
	}
	return s.categories.GetBySlug(ctx, slug)
}

func (s *CatalogService) ListProducts(ctx context.Context, filter domain.ProductFilter) (domain.PageResult[domain.Product], error) {
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

	if normalizedQuery := normalizeSearchQuery(filter.Query); len(normalizedQuery) >= 2 {
		_ = s.products.TrackSearchQuery(ctx, normalizedQuery)
	}

	return result, nil
}

func (s *CatalogService) GetProductByID(ctx context.Context, id uuid.UUID) (domain.Product, error) {
	if id == uuid.Nil {
		return domain.Product{}, domain.ErrInvalidInput
	}
	return s.products.GetByID(ctx, id)
}

func (s *CatalogService) GetProductBySlug(ctx context.Context, slug string) (domain.Product, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return domain.Product{}, domain.ErrInvalidInput
	}
	return s.products.GetBySlug(ctx, slug)
}

func (s *CatalogService) SearchSuggestions(ctx context.Context, query string, limit int) ([]domain.SearchSuggestion, error) {
	query = normalizeSearchQuery(query)
	if len(query) < 2 {
		return []domain.SearchSuggestion{}, nil
	}
	if limit <= 0 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}
	return s.products.SearchSuggestions(ctx, query, limit)
}

func (s *CatalogService) PopularSearches(ctx context.Context, limit int) ([]domain.PopularSearch, error) {
	if limit <= 0 {
		limit = 6
	}
	if limit > 20 {
		limit = 20
	}
	return s.products.ListPopularSearches(ctx, limit)
}

func (s *CatalogService) TrackView(ctx context.Context, userID, productID uuid.UUID) error {
	if userID == uuid.Nil || productID == uuid.Nil {
		return domain.ErrInvalidInput
	}
	return s.events.Create(ctx, userID, productID, domain.ProductEventView)
}

func (s *CatalogService) ListReviews(ctx context.Context, productID uuid.UUID, limit int) ([]domain.Review, error) {
	if productID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.reviews.ListByProductID(ctx, productID, limit)
}

func (s *CatalogService) AddReview(
	ctx context.Context,
	userID, productID uuid.UUID,
	rating int,
	comment string,
) (domain.Review, error) {
	if userID == uuid.Nil || productID == uuid.Nil {
		return domain.Review{}, domain.ErrInvalidInput
	}
	if rating < 1 || rating > 5 {
		return domain.Review{}, domain.ErrInvalidInput
	}

	comment = strings.TrimSpace(comment)
	if len(comment) < 3 || len(comment) > 1500 {
		return domain.Review{}, domain.ErrInvalidInput
	}

	if _, err := s.products.GetByID(ctx, productID); err != nil {
		return domain.Review{}, err
	}

	return s.reviews.Create(ctx, productID, userID, rating, comment)
}

func normalizeSearchQuery(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}
