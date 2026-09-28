package checkout

import "testing"

func TestPromotionLinesFromCheckoutItems(t *testing.T) {
	items := []Item{
		{
			VariantID:       "variant-a",
			Quantity:        2,
			UnitPriceAmount: 1250,
		},
		{
			VariantID:       "variant-b",
			Quantity:        1,
			UnitPriceAmount: 3400,
		},
	}

	lines := promotionLinesFromCheckoutItems(items)

	if len(lines) != 2 {
		t.Fatalf("expected 2 promotion lines, got %d", len(lines))
	}

	if lines[0].VariantID != "variant-a" ||
		lines[0].Quantity != 2 ||
		lines[0].UnitPriceAmount != 1250 {
		t.Fatalf("unexpected first promotion line: %+v", lines[0])
	}

	if lines[1].VariantID != "variant-b" ||
		lines[1].Quantity != 1 ||
		lines[1].UnitPriceAmount != 3400 {
		t.Fatalf("unexpected second promotion line: %+v", lines[1])
	}
}
