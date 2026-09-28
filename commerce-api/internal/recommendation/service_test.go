package recommendation

import (
	"context"
	"errors"
	"testing"
)

type fakeRankingStore struct {
	customerID string
	limit      int
	productIDs []string
	err        error
}

func (f *fakeRankingStore) RankProductIDs(
	_ context.Context,
	customerID string,
	limit int,
) ([]string, error) {
	f.customerID =
		customerID

	f.limit =
		limit

	if f.err != nil {
		return nil, f.err
	}

	return f.productIDs, nil
}

func TestRankProductIDsTrimsCustomerAndUsesDefaultLimit(
	t *testing.T,
) {
	store :=
		&fakeRankingStore{
			productIDs: []string{
				"product-2",
				"product-1",
			},
		}

	service :=
		NewService(
			store,
		)

	result, err :=
		service.RankProductIDs(
			context.Background(),
			"  customer-1  ",
		)
	if err != nil {
		t.Fatalf(
			"rank products: %v",
			err,
		)
	}

	if store.customerID !=
		"customer-1" {
		t.Fatalf(
			"unexpected customer ID %q",
			store.customerID,
		)
	}

	if store.limit !=
		defaultRecommendationLimit {
		t.Fatalf(
			"unexpected limit %d",
			store.limit,
		)
	}

	if len(result) != 2 ||
		result[0] != "product-2" ||
		result[1] != "product-1" {
		t.Fatalf(
			"unexpected result %#v",
			result,
		)
	}
}

func TestRankProductIDsGuestDoesNotHitStore(
	t *testing.T,
) {
	store :=
		&fakeRankingStore{
			err: errors.New(
				"must not be called",
			),
		}

	service :=
		NewService(
			store,
		)

	result, err :=
		service.RankProductIDs(
			context.Background(),
			"   ",
		)
	if err != nil {
		t.Fatalf(
			"guest ranking: %v",
			err,
		)
	}

	if len(result) != 0 {
		t.Fatalf(
			"expected empty guest ranking, got %#v",
			result,
		)
	}

	if store.limit != 0 {
		t.Fatalf(
			"guest ranking called store with limit %d",
			store.limit,
		)
	}
}
