package catalog

import (
	"context"
	"errors"
	"testing"
)

type fakePagedCatalogStore struct {
	query ListProductsQuery
	page  ProductListPage
	err   error
}

func (f *fakePagedCatalogStore) ListActiveProducts(
	_ context.Context,
) ([]ProductCard, error) {
	return []ProductCard{},
		nil
}

func (f *fakePagedCatalogStore) FindActiveProductBySlug(
	_ context.Context,
	_ string,
) (ProductDetail, error) {
	return ProductDetail{},
		errors.New(
			"not implemented in listing test",
		)
}

func (f *fakePagedCatalogStore) ListActiveProductsPage(
	_ context.Context,
	query ListProductsQuery,
) (ProductListPage, error) {
	f.query =
		query

	if f.err != nil {
		return ProductListPage{},
			f.err
	}

	return f.page,
		nil
}

func TestParseListProductsInput(
	t *testing.T,
) {
	input, err :=
		ParseListProductsInput(
			"2",
			"24",
			" shirts ",
			"1000",
			"5000",
			"true",
			"price_asc",
		)
	if err != nil {
		t.Fatalf(
			"parse product listing: %v",
			err,
		)
	}

	if input.Page != 2 ||
		input.Limit != 24 {

		t.Fatalf(
			"unexpected pagination: %#v",
			input,
		)
	}

	if input.CategorySlug !=
		"shirts" {

		t.Fatalf(
			"unexpected category %q",
			input.CategorySlug,
		)
	}

	if input.MinPrice == nil ||
		*input.MinPrice != 1000 {

		t.Fatalf(
			"unexpected min price: %#v",
			input.MinPrice,
		)
	}

	if input.MaxPrice == nil ||
		*input.MaxPrice != 5000 {

		t.Fatalf(
			"unexpected max price: %#v",
			input.MaxPrice,
		)
	}

	if input.InStock == nil ||
		!*input.InStock {

		t.Fatalf(
			"unexpected stock filter: %#v",
			input.InStock,
		)
	}

	if input.Sort !=
		ProductSortPriceAsc {

		t.Fatalf(
			"unexpected sort %q",
			input.Sort,
		)
	}
}

func TestParseListProductsInputRejectsInvalidRange(
	t *testing.T,
) {
	_, err :=
		ParseListProductsInput(
			"",
			"",
			"",
			"5000",
			"1000",
			"",
			"",
		)

	if !errors.Is(
		err,
		ErrInvalidPriceRange,
	) {
		t.Fatalf(
			"expected ErrInvalidPriceRange, got %v",
			err,
		)
	}
}

func TestListProductsForCustomerRanksBeforePageQuery(
	t *testing.T,
) {
	store :=
		&fakePagedCatalogStore{
			page: ProductListPage{
				Items: []ProductCard{
					{
						ID: "product-3",
					},
					{
						ID: "product-1",
					},
				},

				Total: 42,
			},
		}

	ranker :=
		&fakeProductRanker{
			productIDs: []string{
				"product-3",
				"product-1",
			},
		}

	service :=
		NewService(
			store,
		)

	service.SetProductRanker(
		ranker,
	)

	result, err :=
		service.ListProductsForCustomer(
			context.Background(),
			"customer-1",
			ListProductsInput{
				Page:  2,
				Limit: 20,
				Sort:  ProductSortRecommended,
			},
		)
	if err != nil {
		t.Fatalf(
			"list product page: %v",
			err,
		)
	}

	if ranker.calls != 1 {
		t.Fatalf(
			"expected one ranker call, got %d",
			ranker.calls,
		)
	}

	if len(
		store.query.RankedProductIDs,
	) != 2 ||
		store.query.RankedProductIDs[0] !=
			"product-3" ||
		store.query.RankedProductIDs[1] !=
			"product-1" {

		t.Fatalf(
			"ranked IDs were not passed to repository before pagination: %#v",
			store.query.RankedProductIDs,
		)
	}

	if store.query.Pagination.Page !=
		2 ||
		store.query.Pagination.Limit !=
			20 ||
		store.query.Pagination.Offset() !=
			20 {

		t.Fatalf(
			"unexpected repository pagination: %#v",
			store.query.Pagination,
		)
	}

	if result.Meta.Total !=
		42 ||
		result.Meta.TotalPages !=
			3 {

		t.Fatalf(
			"unexpected pagination meta: %#v",
			result.Meta,
		)
	}

	if result.Meta.Recommendation == nil ||
		!result.Meta.Recommendation.Personalized ||
		result.Meta.Recommendation.Strategy != "personalized" {

		t.Fatalf(
			"unexpected recommendation meta: %#v",
			result.Meta.Recommendation,
		)
	}
}

func TestListProductsDeterministicSortSkipsRanker(
	t *testing.T,
) {
	store :=
		&fakePagedCatalogStore{
			page: ProductListPage{
				Items: []ProductCard{},
				Total: 0,
			},
		}

	ranker :=
		&fakeProductRanker{
			productIDs: []string{
				"product-1",
			},
		}

	service :=
		NewService(
			store,
		)

	service.SetProductRanker(
		ranker,
	)

	_, err :=
		service.ListProductsForCustomer(
			context.Background(),
			"customer-1",
			ListProductsInput{
				Sort: ProductSortNewest,
			},
		)
	if err != nil {
		t.Fatalf(
			"list newest products: %v",
			err,
		)
	}

	if ranker.calls != 0 {
		t.Fatalf(
			"deterministic sort called ranker %d times",
			ranker.calls,
		)
	}
}
