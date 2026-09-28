package promotion

import (
	"context"
	"errors"
	"testing"
)

type selectionTestStore struct {
	byCode    map[string]Promotion
	automatic []Promotion
	eligible  map[string]map[string]struct{}
}

func (s *selectionTestStore) GetByCode(
	_ context.Context,
	code string,
) (Promotion, error) {
	item, ok := s.byCode[code]
	if !ok {
		return Promotion{}, ErrPromotionNotFound
	}

	return item, nil
}

func (s *selectionTestStore) ListAutomatic(
	_ context.Context,
) ([]Promotion, error) {
	return append([]Promotion(nil), s.automatic...), nil
}

func (s *selectionTestStore) ResolveEligibleVariantIDs(
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

func TestEvaluateValidCodeDoesNotReplaceBetterAutomaticPromotion(t *testing.T) {
	codeBPS := 1000
	automaticBPS := 2000

	store := &selectionTestStore{
		byCode: map[string]Promotion{
			"SAVE10": {
				ID:            "code-promo",
				Name:          "Code 10%",
				Code:          "SAVE10",
				Scope:         ScopeOrder,
				CampaignType:  CampaignTypeStandard,
				DiscountType:  DiscountTypePercentage,
				PercentageBPS: &codeBPS,
				Currency:      "BDT",
				Status:        StatusActive,
			},
		},
		automatic: []Promotion{
			{
				ID:            "automatic-promo",
				Name:          "Automatic 20%",
				Scope:         ScopeOrder,
				CampaignType:  CampaignTypeStandard,
				DiscountType:  DiscountTypePercentage,
				PercentageBPS: &automaticBPS,
				Currency:      "BDT",
				Status:        StatusActive,
			},
		},
	}

	result, err := NewService(store).Evaluate(
		context.Background(),
		EvaluateInput{
			Code:           "SAVE10",
			SubtotalAmount: 10000,
			Currency:       "BDT",
		},
	)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if result.PromotionID != "automatic-promo" {
		t.Fatalf("promotion id = %q, want automatic-promo", result.PromotionID)
	}

	if result.PromotionCode != "" {
		t.Fatalf("promotion code = %q, want empty automatic code", result.PromotionCode)
	}

	if result.DiscountAmount != 2000 {
		t.Fatalf("discount amount = %d, want 2000", result.DiscountAmount)
	}
}

func TestEvaluateValidCodeWinsWhenItIsBetter(t *testing.T) {
	codeBPS := 3000
	automaticBPS := 2000

	store := &selectionTestStore{
		byCode: map[string]Promotion{
			"SAVE30": {
				ID:            "code-promo",
				Name:          "Code 30%",
				Code:          "SAVE30",
				Scope:         ScopeOrder,
				CampaignType:  CampaignTypeStandard,
				DiscountType:  DiscountTypePercentage,
				PercentageBPS: &codeBPS,
				Currency:      "BDT",
				Status:        StatusActive,
			},
		},
		automatic: []Promotion{
			{
				ID:            "automatic-promo",
				Name:          "Automatic 20%",
				Scope:         ScopeOrder,
				CampaignType:  CampaignTypeStandard,
				DiscountType:  DiscountTypePercentage,
				PercentageBPS: &automaticBPS,
				Currency:      "BDT",
				Status:        StatusActive,
			},
		},
	}

	result, err := NewService(store).Evaluate(
		context.Background(),
		EvaluateInput{
			Code:           "SAVE30",
			SubtotalAmount: 10000,
			Currency:       "BDT",
		},
	)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if result.PromotionID != "code-promo" {
		t.Fatalf("promotion id = %q, want code-promo", result.PromotionID)
	}

	if result.PromotionCode != "SAVE30" {
		t.Fatalf("promotion code = %q, want SAVE30", result.PromotionCode)
	}

	if result.DiscountAmount != 3000 {
		t.Fatalf("discount amount = %d, want 3000", result.DiscountAmount)
	}
}

func TestEvaluateValidCodeWinsExactTie(t *testing.T) {
	bps := 2000

	store := &selectionTestStore{
		byCode: map[string]Promotion{
			"SAVE20": {
				ID:            "code-promo",
				Name:          "Code 20%",
				Code:          "SAVE20",
				Scope:         ScopeOrder,
				CampaignType:  CampaignTypeStandard,
				DiscountType:  DiscountTypePercentage,
				PercentageBPS: &bps,
				Currency:      "BDT",
				Status:        StatusActive,
			},
		},
		automatic: []Promotion{
			{
				ID:            "automatic-promo",
				Name:          "Automatic 20%",
				Scope:         ScopeOrder,
				CampaignType:  CampaignTypeStandard,
				DiscountType:  DiscountTypePercentage,
				PercentageBPS: &bps,
				Currency:      "BDT",
				Status:        StatusActive,
			},
		},
	}

	result, err := NewService(store).Evaluate(
		context.Background(),
		EvaluateInput{
			Code:           "SAVE20",
			SubtotalAmount: 10000,
			Currency:       "BDT",
		},
	)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if result.PromotionID != "code-promo" {
		t.Fatalf("promotion id = %q, want code-promo", result.PromotionID)
	}

	if result.PromotionCode != "SAVE20" {
		t.Fatalf("promotion code = %q, want SAVE20", result.PromotionCode)
	}
}

func TestEvaluateInvalidExplicitCodeDoesNotFallBackToAutomatic(t *testing.T) {
	bps := 2500

	store := &selectionTestStore{
		byCode: map[string]Promotion{},
		automatic: []Promotion{
			{
				ID:            "automatic-promo",
				Name:          "Automatic 25%",
				Scope:         ScopeOrder,
				CampaignType:  CampaignTypeStandard,
				DiscountType:  DiscountTypePercentage,
				PercentageBPS: &bps,
				Currency:      "BDT",
				Status:        StatusActive,
			},
		},
	}

	_, err := NewService(store).Evaluate(
		context.Background(),
		EvaluateInput{
			Code:           "DOES_NOT_EXIST",
			SubtotalAmount: 10000,
			Currency:       "BDT",
		},
	)
	if !errors.Is(err, ErrPromotionNotFound) {
		t.Fatalf("error = %v, want ErrPromotionNotFound", err)
	}
}
