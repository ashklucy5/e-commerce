package promotion

import (
	"context"
	"testing"
	"time"
)

type targetTestStore struct {
	byCode    map[string]Promotion
	automatic []Promotion
	eligible  map[string]map[string]struct{}
}

func (s *targetTestStore) GetByCode(
	_ context.Context,
	code string,
) (Promotion, error) {
	promotion, ok := s.byCode[code]
	if !ok {
		return Promotion{}, ErrPromotionNotFound
	}

	return promotion, nil
}

func (s *targetTestStore) ListAutomatic(
	_ context.Context,
) ([]Promotion, error) {
	return append([]Promotion(nil), s.automatic...), nil
}

func (s *targetTestStore) ResolveEligibleVariantIDs(
	_ context.Context,
	promotionID string,
	variantIDs []string,
) (map[string]struct{}, error) {
	allowed := s.eligible[promotionID]
	result := make(map[string]struct{})

	for _, variantID := range variantIDs {
		if _, ok := allowed[variantID]; ok {
			result[variantID] = struct{}{}
		}
	}

	return result, nil
}

func TestEvaluateProductPromotionUsesOnlyEligibleLines(t *testing.T) {
	bps := 2000

	store := &targetTestStore{
		automatic: []Promotion{
			{
				ID:            "promo-product",
				Name:          "Targeted 20%",
				Scope:         ScopeProduct,
				CampaignType:  CampaignTypeStandard,
				DiscountType:  DiscountTypePercentage,
				PercentageBPS: &bps,
				Currency:      "BDT",
				Status:        StatusActive,
			},
		},
		eligible: map[string]map[string]struct{}{
			"promo-product": {
				"variant-a": {},
			},
		},
	}

	service := NewService(store)

	result, err := service.Evaluate(
		context.Background(),
		EvaluateInput{
			SubtotalAmount: 2500,
			Currency:       "BDT",
			Lines: []EvaluateLine{
				{
					VariantID:       "variant-a",
					Quantity:        2,
					UnitPriceAmount: 1000,
				},
				{
					VariantID:       "variant-b",
					Quantity:        1,
					UnitPriceAmount: 500,
				},
			},
			Now: time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC),
		},
	)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if !result.Applied {
		t.Fatal("expected promotion to apply")
	}

	if result.DiscountAmount != 400 {
		t.Fatalf(
			"discount amount = %d, want 400",
			result.DiscountAmount,
		)
	}
}

func TestEvaluateProductFixedDiscountAppliesPerEligibleUnit(t *testing.T) {
	fixed := int64(125)

	store := &targetTestStore{
		automatic: []Promotion{
			{
				ID:           "promo-fixed",
				Name:         "Fixed item discount",
				Scope:        ScopeProduct,
				CampaignType: CampaignTypeStandard,
				DiscountType: DiscountTypeFixed,
				FixedAmount:  &fixed,
				Currency:     "BDT",
				Status:       StatusActive,
			},
		},
		eligible: map[string]map[string]struct{}{
			"promo-fixed": {
				"variant-a": {},
			},
		},
	}

	service := NewService(store)

	result, err := service.Evaluate(
		context.Background(),
		EvaluateInput{
			SubtotalAmount: 3000,
			Currency:       "BDT",
			Lines: []EvaluateLine{
				{
					VariantID:       "variant-a",
					Quantity:        3,
					UnitPriceAmount: 1000,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if result.DiscountAmount != 375 {
		t.Fatalf(
			"discount amount = %d, want 375",
			result.DiscountAmount,
		)
	}
}

func TestEvaluateProductPromotionUsesEligibleSubtotalForMinimum(t *testing.T) {
	bps := 1000

	store := &targetTestStore{
		automatic: []Promotion{
			{
				ID:                    "promo-minimum",
				Name:                  "Eligible minimum",
				Scope:                 ScopeProduct,
				CampaignType:          CampaignTypeStandard,
				DiscountType:          DiscountTypePercentage,
				PercentageBPS:         &bps,
				MinimumSubtotalAmount: 1500,
				Currency:              "BDT",
				Status:                StatusActive,
			},
		},
		eligible: map[string]map[string]struct{}{
			"promo-minimum": {
				"variant-a": {},
			},
		},
	}

	service := NewService(store)

	result, err := service.Evaluate(
		context.Background(),
		EvaluateInput{
			SubtotalAmount: 5000,
			Currency:       "BDT",
			Lines: []EvaluateLine{
				{
					VariantID:       "variant-a",
					Quantity:        1,
					UnitPriceAmount: 1000,
				},
				{
					VariantID:       "variant-b",
					Quantity:        1,
					UnitPriceAmount: 4000,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if result.Applied {
		t.Fatalf("expected product promotion to be ineligible, got %+v", result)
	}
}

func TestEvaluateAutomaticChoosesLargestDiscountAcrossScopes(t *testing.T) {
	orderBPS := 1000
	productBPS := 2500

	store := &targetTestStore{
		automatic: []Promotion{
			{
				ID:            "order-promo",
				Name:          "Order 10%",
				Scope:         ScopeOrder,
				CampaignType:  CampaignTypeStandard,
				DiscountType:  DiscountTypePercentage,
				PercentageBPS: &orderBPS,
				Currency:      "BDT",
				Status:        StatusActive,
			},
			{
				ID:            "flash-promo",
				Name:          "Flash 25%",
				Scope:         ScopeProduct,
				CampaignType:  CampaignTypeFlashSale,
				DiscountType:  DiscountTypePercentage,
				PercentageBPS: &productBPS,
				Currency:      "BDT",
				Status:        StatusActive,
			},
		},
		eligible: map[string]map[string]struct{}{
			"flash-promo": {
				"variant-a": {},
			},
		},
	}

	service := NewService(store)

	result, err := service.Evaluate(
		context.Background(),
		EvaluateInput{
			SubtotalAmount: 5000,
			Currency:       "BDT",
			Lines: []EvaluateLine{
				{
					VariantID:       "variant-a",
					Quantity:        1,
					UnitPriceAmount: 3000,
				},
				{
					VariantID:       "variant-b",
					Quantity:        1,
					UnitPriceAmount: 2000,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if result.PromotionID != "flash-promo" {
		t.Fatalf(
			"promotion id = %q, want flash-promo",
			result.PromotionID,
		)
	}

	if result.DiscountAmount != 750 {
		t.Fatalf(
			"discount amount = %d, want 750",
			result.DiscountAmount,
		)
	}
}
