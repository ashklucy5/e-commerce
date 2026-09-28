package validation

import (
	"testing"
)

func TestValidatorCollectsFieldErrors(t *testing.T) {
	validator := New()

	validator.Required(
		"name",
		"",
	)

	validator.UUID(
		"product_id",
		"not-a-uuid",
	)

	validator.OneOf(
		"status",
		"unknown",
		"active",
		"inactive",
	)

	err := validator.Err()
	if err == nil {
		t.Fatal(
			"expected validation error",
		)
	}

	validationErrors, ok := err.(*Errors)
	if !ok {
		t.Fatalf(
			"expected *Errors, got %T",
			err,
		)
	}

	if validationErrors.Len() != 3 {
		t.Fatalf(
			"expected 3 field errors, got %d",
			validationErrors.Len(),
		)
	}
}

func TestIsUUID(t *testing.T) {
	const valid = "93ef63dd-8421-4887-a49c-7784570e167d"

	if !IsUUID(
		valid,
	) {
		t.Fatal(
			"expected UUID to be valid",
		)
	}

	if IsUUID(
		"93ef63dd-8421",
	) {
		t.Fatal(
			"expected malformed UUID to be invalid",
		)
	}
}
