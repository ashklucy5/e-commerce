package admin

import (
	"errors"
	"testing"

	promotiondomain "project.local/commerce-api/internal/promotion"
)

func TestNormalizeAdminProductDiscountPercentage(t *testing.T) {
	bps := 1500

	result, err := normalizeAdminProductDiscountInput(
		ProductDiscountConfigInput{
			DiscountType:  " Percentage ",
			PercentageBPS: &bps,
		},
	)
	if err != nil {
		t.Fatalf("normalizeAdminProductDiscountInput returned error: %v", err)
	}

	if result.DiscountType != promotiondomain.DiscountTypePercentage {
		t.Fatalf(
			"discount type = %q, want %q",
			result.DiscountType,
			promotiondomain.DiscountTypePercentage,
		)
	}
}

func TestNormalizeAdminProductDiscountRejectsMixedConfiguration(t *testing.T) {
	bps := 1500
	fixed := int64(500)

	_, err := normalizeAdminProductDiscountInput(
		ProductDiscountConfigInput{
			DiscountType:  promotiondomain.DiscountTypePercentage,
			PercentageBPS: &bps,
			FixedAmount:   &fixed,
		},
	)
	if !errors.Is(err, ErrInvalidAdminProductDiscount) {
		t.Fatalf("error = %v, want ErrInvalidAdminProductDiscount", err)
	}
}
