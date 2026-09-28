package catalog

import (
	"context"
	"errors"
	"testing"
)

type fakeCatalogStore struct {
	products []ProductCard
	err      error
}

func (f *fakeCatalogStore) ListActiveProducts(
	_ context.Context,
) ([]ProductCard, error) {
	if f.err != nil {
		return nil, f.err
	}

	return append(
		[]ProductCard(nil),
		f.products...,
	), nil
}

func (f *fakeCatalogStore) FindActiveProductBySlug(
	_ context.Context,
	_ string,
) (ProductDetail, error) {
	return ProductDetail{},
		errors.New(
			"not implemented in this test",
		)
}

type fakeProductRanker struct {
	customerID string
	productIDs []string
	err        error
	calls      int
}

func (f *fakeProductRanker) RankProductIDs(
	_ context.Context,
	customerID string,
) ([]string, error) {
	f.calls++
	f.customerID =
		customerID

	if f.err != nil {
		return nil, f.err
	}

	return f.productIDs, nil
}

func TestListActiveProductsForCustomerReordersFeed(
	t *testing.T,
) {
	store :=
		&fakeCatalogStore{
			products: []ProductCard{
				{
					ID:   "product-1",
					Name: "One",
				},
				{
					ID:   "product-2",
					Name: "Two",
				},
				{
					ID:   "product-3",
					Name: "Three",
				},
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
		service.ListActiveProductsForCustomer(
			context.Background(),
			"customer-1",
		)
	if err != nil {
		t.Fatalf(
			"list personalized catalog: %v",
			err,
		)
	}

	if ranker.calls != 1 ||
		ranker.customerID !=
			"customer-1" {
		t.Fatalf(
			"unexpected ranker call count=%d customer=%q",
			ranker.calls,
			ranker.customerID,
		)
	}

	expected :=
		[]string{
			"product-3",
			"product-1",
			"product-2",
		}

	if len(result) !=
		len(expected) {
		t.Fatalf(
			"unexpected result length %d",
			len(result),
		)
	}

	for index := range expected {
		if result[index].ID !=
			expected[index] {
			t.Fatalf(
				"position %d expected %q got %q",
				index,
				expected[index],
				result[index].ID,
			)
		}
	}
}

func TestListActiveProductsGuestKeepsDefaultOrder(
	t *testing.T,
) {
	store :=
		&fakeCatalogStore{
			products: []ProductCard{
				{
					ID: "product-1",
				},
				{
					ID: "product-2",
				},
			},
		}

	ranker :=
		&fakeProductRanker{
			productIDs: []string{
				"product-2",
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
		service.ListActiveProductsForCustomer(
			context.Background(),
			"",
		)
	if err != nil {
		t.Fatalf(
			"list guest catalog: %v",
			err,
		)
	}

	if ranker.calls != 0 {
		t.Fatalf(
			"guest feed called ranker %d times",
			ranker.calls,
		)
	}

	if result[0].ID !=
		"product-1" ||
		result[1].ID !=
			"product-2" {
		t.Fatalf(
			"guest order changed: %#v",
			result,
		)
	}
}

func TestListActiveProductsRankingFailureFallsBack(
	t *testing.T,
) {
	store :=
		&fakeCatalogStore{
			products: []ProductCard{
				{
					ID: "product-1",
				},
				{
					ID: "product-2",
				},
			},
		}

	ranker :=
		&fakeProductRanker{
			err: errors.New(
				"ranking database unavailable",
			),
		}

	service :=
		NewService(
			store,
		)

	service.SetProductRanker(
		ranker,
	)

	result, err :=
		service.ListActiveProductsForCustomer(
			context.Background(),
			"customer-1",
		)
	if err != nil {
		t.Fatalf(
			"ranking failure must not fail catalog: %v",
			err,
		)
	}

	if result[0].ID !=
		"product-1" ||
		result[1].ID !=
			"product-2" {
		t.Fatalf(
			"fallback order changed: %#v",
			result,
		)
	}
}
