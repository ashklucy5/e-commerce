package promotion

import (
	"context"
	"strings"
	"testing"
	"time"
)

type flashSaleConflictTestStore struct {
	records []flashSaleTargetRecord
}

func (s *flashSaleConflictTestStore) GetByCode(
	_ context.Context,
	_ string,
) (Promotion, error) {
	return Promotion{}, ErrPromotionNotFound
}

func (s *flashSaleConflictTestStore) ListAutomatic(
	_ context.Context,
) ([]Promotion, error) {
	return nil, nil
}

func (s *flashSaleConflictTestStore) ResolveEligibleVariantIDs(
	_ context.Context,
	_ string,
	_ []string,
) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

func (s *flashSaleConflictTestStore) ListActiveFlashSaleTargets(
	_ context.Context,
	_ string,
	_ time.Time,
) ([]flashSaleTargetRecord, error) {
	return append([]flashSaleTargetRecord(nil), s.records...), nil
}

func TestListActiveFlashSalesRejectsVariantInMultipleCampaigns(t *testing.T) {
	bps := 2000
	startsAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	endsAt := startsAt.Add(2 * time.Hour)

	base := flashSaleTargetRecord{
		Promotion: Promotion{
			Scope:         ScopeProduct,
			CampaignType:  CampaignTypeFlashSale,
			DiscountType:  DiscountTypePercentage,
			PercentageBPS: &bps,
			Currency:      "BDT",
			Status:        StatusActive,
			StartsAt:      &startsAt,
			EndsAt:        &endsAt,
		},
		ProductID:   "product-a",
		ProductName: "Product A",
		ProductSlug: "product-a",
		VariantID:   "variant-a",
		SKU:         "SKU-A",
		PriceAmount: 1000,
		Currency:    "BDT",
	}

	first := base
	first.Promotion.ID = "promotion-a"
	first.Promotion.Name = "Flash A"

	second := base
	second.Promotion.ID = "promotion-b"
	second.Promotion.Name = "Flash B"

	service := NewService(
		&flashSaleConflictTestStore{
			records: []flashSaleTargetRecord{
				first,
				second,
			},
		},
	)

	_, err := service.ListActiveFlashSales(
		context.Background(),
		"BDT",
		startsAt.Add(time.Hour),
	)
	if err == nil {
		t.Fatal("expected overlapping flash-sale target error")
	}

	if !strings.Contains(err.Error(), "active flash sale target conflict") {
		t.Fatalf("error = %v, want active flash sale target conflict", err)
	}
}
