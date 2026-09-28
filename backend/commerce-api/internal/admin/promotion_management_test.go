package admin

import (
	"errors"
	"testing"
	"time"

	promotiondomain "project.local/commerce-api/internal/promotion"
)

func TestNormalizeAdminPromotionDefaultsExistingOrderPromotionShape(t *testing.T) {
	bps := 1000

	result, err := normalizeAdminPromotionInput(
		PromotionConfigInput{
			Name:          "Welcome",
			DiscountType:  promotiondomain.DiscountTypePercentage,
			PercentageBPS: &bps,
			Currency:      "bdt",
			Status:        promotiondomain.StatusDraft,
		},
	)
	if err != nil {
		t.Fatalf("normalizeAdminPromotionInput returned error: %v", err)
	}

	if result.Scope != promotiondomain.ScopeOrder {
		t.Fatalf("scope = %q, want order", result.Scope)
	}

	if result.CampaignType != promotiondomain.CampaignTypeStandard {
		t.Fatalf(
			"campaign_type = %q, want standard",
			result.CampaignType,
		)
	}
}

func TestNormalizeAdminPromotionAcceptsFlashSaleWithTargets(t *testing.T) {
	bps := 2500
	startsAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	endsAt := startsAt.Add(2 * time.Hour)

	result, err := normalizeAdminPromotionInput(
		PromotionConfigInput{
			Name:          "Two hour flash sale",
			Scope:         promotiondomain.ScopeProduct,
			CampaignType:  promotiondomain.CampaignTypeFlashSale,
			DiscountType:  promotiondomain.DiscountTypePercentage,
			PercentageBPS: &bps,
			Currency:      "BDT",
			Status:        promotiondomain.StatusActive,
			StartsAt:      &startsAt,
			EndsAt:        &endsAt,
			Targets: []PromotionTargetInput{
				{
					ProductID: "11111111-1111-4111-8111-111111111111",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("normalizeAdminPromotionInput returned error: %v", err)
	}

	if len(result.Targets) != 1 {
		t.Fatalf("target count = %d, want 1", len(result.Targets))
	}
}

func TestNormalizeAdminPromotionRejectsFlashSaleCode(t *testing.T) {
	bps := 2500
	code := "FLASH25"
	startsAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	endsAt := startsAt.Add(2 * time.Hour)

	_, err := normalizeAdminPromotionInput(
		PromotionConfigInput{
			Name:          "Invalid flash sale",
			Code:          &code,
			Scope:         promotiondomain.ScopeProduct,
			CampaignType:  promotiondomain.CampaignTypeFlashSale,
			DiscountType:  promotiondomain.DiscountTypePercentage,
			PercentageBPS: &bps,
			Currency:      "BDT",
			Status:        promotiondomain.StatusActive,
			StartsAt:      &startsAt,
			EndsAt:        &endsAt,
			Targets: []PromotionTargetInput{
				{
					VariantID: "22222222-2222-4222-8222-222222222222",
				},
			},
		},
	)
	if !errors.Is(err, ErrInvalidAdminPromotion) {
		t.Fatalf("error = %v, want ErrInvalidAdminPromotion", err)
	}
}

func TestNormalizeAdminPromotionRequiresTargetsForActiveProductScope(t *testing.T) {
	fixed := int64(500)

	_, err := normalizeAdminPromotionInput(
		PromotionConfigInput{
			Name:         "Targeted coupon",
			Scope:        promotiondomain.ScopeProduct,
			CampaignType: promotiondomain.CampaignTypeStandard,
			DiscountType: promotiondomain.DiscountTypeFixed,
			FixedAmount:  &fixed,
			Currency:     "BDT",
			Status:       promotiondomain.StatusActive,
		},
	)
	if !errors.Is(err, ErrInvalidAdminPromotion) {
		t.Fatalf("error = %v, want ErrInvalidAdminPromotion", err)
	}
}

func TestNormalizeAdminPromotionRejectsFlashSaleMaximumDiscount(t *testing.T) {
	bps := 2500
	maximum := int64(500)
	startsAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	endsAt := startsAt.Add(2 * time.Hour)

	_, err := normalizeAdminPromotionInput(
		PromotionConfigInput{
			Name:                  "Ambiguous flash sale cap",
			Scope:                 promotiondomain.ScopeProduct,
			CampaignType:          promotiondomain.CampaignTypeFlashSale,
			DiscountType:          promotiondomain.DiscountTypePercentage,
			PercentageBPS:         &bps,
			MaximumDiscountAmount: &maximum,
			Currency:              "BDT",
			Status:                promotiondomain.StatusActive,
			StartsAt:              &startsAt,
			EndsAt:                &endsAt,
			Targets: []PromotionTargetInput{
				{
					ProductID: "33333333-3333-4333-8333-333333333333",
				},
			},
		},
	)
	if !errors.Is(err, ErrInvalidAdminPromotion) {
		t.Fatalf("error = %v, want ErrInvalidAdminPromotion", err)
	}
}
