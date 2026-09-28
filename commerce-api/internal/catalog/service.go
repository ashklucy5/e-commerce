package catalog

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/platform/pagination"
)

type Store interface {
	ListActiveProducts(
		ctx context.Context,
	) ([]ProductCard, error)

	FindActiveProductBySlug(
		ctx context.Context,
		slug string,
	) (ProductDetail, error)
}

type PageStore interface {
	ListActiveProductsPage(
		ctx context.Context,
		input ListProductsQuery,
	) (ProductListPage, error)
}

type ProductRanker interface {
	RankProductIDs(
		ctx context.Context,
		customerID string,
	) ([]string, error)
}

// Service contains customer-facing catalog business logic.
type Service struct {
	repository     Store
	pageRepository PageStore
	ranker         ProductRanker
}

// NewService creates a catalog service.
func NewService(
	repository Store,
) *Service {
	service :=
		&Service{
			repository: repository,
		}

	if pageRepository, ok :=
		repository.(PageStore); ok {

		service.pageRepository =
			pageRepository
	}

	return service
}

func (s *Service) SetProductRanker(
	ranker ProductRanker,
) {
	s.ranker =
		ranker
}

// ListProductsForCustomer returns one server-paginated product page.
//
// Recommendation ordering is resolved before database LIMIT/OFFSET.
// This prevents us from paginating first and then only rearranging the
// products that happened to land on that page.
func (s *Service) ListProductsForCustomer(
	ctx context.Context,
	customerID string,
	input ListProductsInput,
) (ProductListResult, error) {
	input, params, err :=
		normalizeListProductsInput(
			input,
		)
	if err != nil {
		return ProductListResult{},
			err
	}

	if s.pageRepository == nil {
		return ProductListResult{},
			fmt.Errorf(
				"paginated catalog repository is unavailable",
			)
	}

	customerID =
		strings.TrimSpace(
			customerID,
		)

	rankedIDs :=
		make(
			[]string,
			0,
		)

	if input.Sort ==
		ProductSortRecommended &&
		customerID != "" &&
		s.ranker != nil {

		rankedIDs, err =
			s.ranker.RankProductIDs(
				ctx,
				customerID,
			)
		if err != nil {
			log.Printf(
				"catalog recommendation ranking failed: %v",
				err,
			)

			rankedIDs =
				make(
					[]string,
					0,
				)
		}
	}

	page, err :=
		s.pageRepository.ListActiveProductsPage(
			ctx,
			ListProductsQuery{
				Pagination: params,

				CategorySlug: input.CategorySlug,

				MinPrice: input.MinPrice,

				MaxPrice: input.MaxPrice,

				InStock: input.InStock,

				Sort: input.Sort,

				RankedProductIDs: rankedIDs,
			},
		)
	if err != nil {
		return ProductListResult{},
			fmt.Errorf(
				"list active product page: %w",
				err,
			)
	}

	if page.Items == nil {
		page.Items =
			make(
				[]ProductCard,
				0,
			)
	}

	var recommendationMeta *RecommendationMeta
	if input.Sort == ProductSortRecommended {
		strategy := "merchandising"
		personalized := false

		if customerID != "" && len(rankedIDs) > 0 {
			strategy = "personalized"
			personalized = true
		}

		recommendationMeta = &RecommendationMeta{
			Strategy:     strategy,
			Personalized: personalized,
		}
	}

	return ProductListResult{
		Items: page.Items,

		Meta: ProductListMeta{
			Meta: pagination.NewMeta(
				params,
				page.Total,
			),

			Sort: input.Sort,

			Recommendation: recommendationMeta,
		},
	}, nil
}

// ListActiveProducts preserves the previous internal unpaginated
// service path. HTTP listing now uses ListProductsForCustomer.
func (s *Service) ListActiveProducts(
	ctx context.Context,
) ([]ProductCard, error) {
	return s.listActiveProducts(
		ctx,
		"",
	)
}

// ListActiveProductsForCustomer preserves the existing
// recommendation-aware behavior for older internal callers/tests.
func (s *Service) ListActiveProductsForCustomer(
	ctx context.Context,
	customerID string,
) ([]ProductCard, error) {
	return s.listActiveProducts(
		ctx,
		strings.TrimSpace(
			customerID,
		),
	)
}

func (s *Service) listActiveProducts(
	ctx context.Context,
	customerID string,
) ([]ProductCard, error) {
	products, err :=
		s.repository.ListActiveProducts(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list active products: %w",
				err,
			)
	}

	if products == nil {
		products =
			make(
				[]ProductCard,
				0,
			)
	}

	if customerID == "" ||
		s.ranker == nil ||
		len(products) < 2 {

		return products,
			nil
	}

	rankedIDs, err :=
		s.ranker.RankProductIDs(
			ctx,
			customerID,
		)
	if err != nil {
		log.Printf(
			"catalog recommendation ranking failed: %v",
			err,
		)

		return products,
			nil
	}

	if len(rankedIDs) == 0 {
		return products,
			nil
	}

	return reorderProductCards(
		products,
		rankedIDs,
	), nil
}

func reorderProductCards(
	products []ProductCard,
	rankedIDs []string,
) []ProductCard {
	byID :=
		make(
			map[string]ProductCard,
			len(products),
		)

	for _, product := range products {

		byID[product.ID] =
			product
	}

	result :=
		make(
			[]ProductCard,
			0,
			len(products),
		)

	seen :=
		make(
			map[string]struct{},
			len(products),
		)

	for _, productID := range rankedIDs {

		product, exists :=
			byID[productID]

		if !exists {
			continue
		}

		if _, exists :=
			seen[productID]; exists {

			continue
		}

		result =
			append(
				result,
				product,
			)

		seen[productID] =
			struct{}{}
	}

	for _, product := range products {

		if _, exists :=
			seen[product.ID]; exists {

			continue
		}

		result =
			append(
				result,
				product,
			)
	}

	return result
}

// GetActiveProductBySlug returns one customer-visible product.
func (s *Service) GetActiveProductBySlug(
	ctx context.Context,
	slug string,
) (ProductDetail, error) {
	slug =
		strings.TrimSpace(
			slug,
		)

	if slug == "" {
		return ProductDetail{},
			ErrInvalidProductSlug
	}

	product, err :=
		s.repository.FindActiveProductBySlug(
			ctx,
			slug,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ProductDetail{},
				ErrProductNotFound
		}

		return ProductDetail{},
			fmt.Errorf(
				"get active product by slug: %w",
				err,
			)
	}

	if err :=
		s.attachImmersiveMedia(
			ctx,
			&product,
		); err != nil {

		return ProductDetail{},
			fmt.Errorf(
				"load product immersive media: %w",
				err,
			)
	}

	return product,
		nil
}
