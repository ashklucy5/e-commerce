package promotion

import (
	"context"
	"testing"
	"time"
)

type storefrontTestStore struct {
	promotions []Promotion
	currency   string
	now        time.Time
}

func (s *storefrontTestStore) GetByCode(
	_ context.Context,
	_ string,
) (Promotion, error) {
	return Promotion{}, ErrPromotionNotFound
}

func (s *storefrontTestStore) ListAutomatic(
	_ context.Context,
) ([]Promotion, error) {
	return nil, nil
}

func (s *storefrontTestStore) ResolveEligibleVariantIDs(
	_ context.Context,
	_ string,
	_ []string,
) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

func (s *storefrontTestStore) ListStorefrontActive(
	_ context.Context,
	currency string,
	now time.Time,
) ([]Promotion, error) {
	s.currency = currency
	s.now = now

	return append([]Promotion(nil), s.promotions...), nil
}

func TestListStorefrontActiveDefaultsCurrencyAndMapsPromotion(t *testing.T) {
	bps := 1500
	startsAt := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	endsAt := startsAt.Add(12 * time.Hour)

	store := &storefrontTestStore{
		promotions: []Promotion{
			{
				ID:                    "promotion-a",
				Name:                  "Automatic 15%",
				Scope:                 ScopeOrder,
				CampaignType:          CampaignTypeStandard,
				DiscountType:          DiscountTypePercentage,
				PercentageBPS:         &bps,
				MinimumSubtotalAmount: 1000,
				Currency:              "BDT",
				Status:                StatusActive,
				StartsAt:              &startsAt,
				EndsAt:                &endsAt,
			},
		},
	}

	service := NewService(store)

	result, err := service.ListStorefrontActive(
		context.Background(),
		"",
		time.Time{},
	)
	if err != nil {
		t.Fatalf("ListStorefrontActive returned error: %v", err)
	}

	if store.currency != "BDT" {
		t.Fatalf("repository currency = %q, want BDT", store.currency)
	}

	if store.now.IsZero() {
		t.Fatal("expected non-zero repository time")
	}

	if len(result) != 1 {
		t.Fatalf("result length = %d, want 1", len(result))
	}

	got := result[0]
	if got.ID != "promotion-a" ||
		got.Scope != ScopeOrder ||
		got.CampaignType != CampaignTypeStandard ||
		got.DiscountType != DiscountTypePercentage ||
		got.PercentageBPS == nil ||
		*got.PercentageBPS != 1500 ||
		got.MinimumSubtotalAmount != 1000 ||
		got.Currency != "BDT" {
		t.Fatalf("unexpected storefront promotion: %+v", got)
	}
}

func TestListStorefrontActiveRejectsInvalidCurrency(t *testing.T) {
	service := NewService(&storefrontTestStore{})

	_, err := service.ListStorefrontActive(
		context.Background(),
		"BD",
		time.Time{},
	)
	if err != ErrInvalidCurrency {
		t.Fatalf("error = %v, want ErrInvalidCurrency", err)
	}
}
