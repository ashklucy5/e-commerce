package order

import "testing"

func TestPromotionLinesFromOrderItems(t *testing.T) {
	items := []checkoutItemSnapshot{
		{
			VariantID:       "variant-a",
			Quantity:        3,
			UnitPriceAmount: 900,
		},
		{
			VariantID:       "variant-b",
			Quantity:        2,
			UnitPriceAmount: 1750,
		},
	}

	lines := promotionLinesFromOrderItems(items)

	if len(lines) != 2 {
		t.Fatalf("expected 2 promotion lines, got %d", len(lines))
	}

	if lines[0].VariantID != "variant-a" ||
		lines[0].Quantity != 3 ||
		lines[0].UnitPriceAmount != 900 {
		t.Fatalf("unexpected first promotion line: %+v", lines[0])
	}

	if lines[1].VariantID != "variant-b" ||
		lines[1].Quantity != 2 ||
		lines[1].UnitPriceAmount != 1750 {
		t.Fatalf("unexpected second promotion line: %+v", lines[1])
	}
}
