package catalogwrite

import (
	"errors"
	"testing"
)

func TestValidateCreateCategoryInputRejectsNegativeSortOrder(
	t *testing.T,
) {
	err := validateCreateCategoryInput(
		CreateCategoryInput{
			Name:      "Accessories",
			SortOrder: -1,
		},
	)

	if !errors.Is(
		err,
		ErrInvalidCategorySortOrder,
	) {
		t.Fatalf(
			"expected ErrInvalidCategorySortOrder, got %v",
			err,
		)
	}
}
