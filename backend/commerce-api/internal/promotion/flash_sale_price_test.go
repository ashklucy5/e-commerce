package promotion

import (
	"context"
	"testing"
	"time"
)

type flashSalePriceTestStore struct {
	records []flashSaleTargetRecord
}

func (s *flashSalePriceTestStore) GetByCode(
	_ context.Context,
	_ string,
) (Promotion, error) {
	return Promotion{}, ErrPromotionNotFound
}

func (s *flashSalePriceTestStore) ListAutomatic(
	_ context.Context,
) ([]Promotion, error) {
	return nil, nil
}

func (s *flashSalePriceTestStore) ResolveEligibleVariantIDs(
	_ context.Context,
	_ string,
	_ []string,
) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

func (s *flashSalePriceTestStore) ListActiveFlashSaleTargets(
	_ context.Context,
	_ string,
	_ time.Time,
) ([]flashSaleTargetRecord, error) {
	return append([]flashSaleTargetRecord(nil), s.records...), nil
}

func TestListActiveFlashSalesCalculatesEffectiveUnitPrice(t *testing.T) {
	bps := 2000
	compareAt := int64(1200)
	startsAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	endsAt := startsAt.Add(2 * time.Hour)

	store := &flashSalePriceTestStore{
		records: []flashSaleTargetRecord{
			{
				Promotion: Promotion{
					ID:            "promotion-a",
					Name:          "Flash 20%",
					Scope:         ScopeProduct,
					CampaignType:  CampaignTypeFlashSale,
					DiscountType:  DiscountTypePercentage,
					PercentageBPS: &bps,
					Currency:      "BDT",
					Status:        StatusActive,
					StartsAt:      &startsAt,
					EndsAt:        &endsAt,
				},
				ProductID:            "product-a",
				ProductName:          "Product A",
				ProductSlug:          "product-a",
				VariantID:            "variant-a",
				SKU:                  "SKU-A",
				PriceAmount:          1000,
				CompareAtPriceAmount: &compareAt,
				Currency:             "BDT",
				PrimaryImageURL:      "https://example.test/product-a.jpg",
			},
		},
	}

	service := NewService(store)

	result, err := service.ListActiveFlashSales(
		context.Background(),
		"BDT",
		startsAt.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("ListActiveFlashSales returned error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("campaign count = %d, want 1", len(result))
	}

	if len(result[0].Targets) != 1 {
		t.Fatalf("target count = %d, want 1", len(result[0].Targets))
	}

	target := result[0].Targets[0]
	if target.PriceAmount != 1000 {
		t.Fatalf("price amount = %d, want 1000", target.PriceAmount)
	}
	if target.DiscountAmount != 200 {
		t.Fatalf("discount amount = %d, want 200", target.DiscountAmount)
	}
	if target.EffectivePriceAmount != 800 {
		t.Fatalf("effective price = %d, want 800", target.EffectivePriceAmount)
	}
	if target.CompareAtPriceAmount == nil || *target.CompareAtPriceAmount != 1200 {
		t.Fatalf("compare-at price = %v, want 1200", target.CompareAtPriceAmount)
	}
}
