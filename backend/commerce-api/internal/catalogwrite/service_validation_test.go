package catalogwrite

import (
	"strings"
	"testing"
)

func TestNormalizeProductInputDefaultsOrderIncrementToOne(
	t *testing.T,
) {
	input := validProductInputForValidation(
		0,
	)

	normalizeProductInput(
		&input,
	)

	if got := input.Variants[0].OrderIncrement; got != 1 {
		t.Fatalf(
			"expected omitted order increment to default to 1, got %d",
			got,
		)
	}

	if err := validateProductInput(
		input,
	); err != nil {
		t.Fatalf(
			"expected normalized product input to be valid, got %v",
			err,
		)
	}
}

func TestValidateProductInputRejectsOrderIncrementOtherThanOne(
	t *testing.T,
) {
	input := validProductInputForValidation(
		5,
	)

	normalizeProductInput(
		&input,
	)

	err := validateProductInput(
		input,
	)
	if err == nil {
		t.Fatal(
			"expected order increment 5 to be rejected",
		)
	}

	if !strings.Contains(
		err.Error(),
		"order increment must be 1",
	) {
		t.Fatalf(
			"unexpected validation error: %v",
			err,
		)
	}
}

func validProductInputForValidation(
	orderIncrement int,
) ProductInput {
	return ProductInput{
		Name:       "Validation Product",
		CategoryID: "category-id",
		Status:     "draft",
		Variants: []VariantInput{
			{
				MinimumOrderQuantity: 1,
				OrderIncrement:       orderIncrement,
				PriceAmount:          100,
				Currency:             "BDT",
			},
		},
	}
}
