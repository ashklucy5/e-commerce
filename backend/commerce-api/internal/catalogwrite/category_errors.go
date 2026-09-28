package catalogwrite

import "errors"

var (
	ErrCategoryNameRequired = errors.New(
		"category name is required",
	)

	ErrCategoryParentNotFound = errors.New(
		"parent category not found",
	)

	ErrInvalidCategorySortOrder = errors.New(
		"sort_order cannot be negative",
	)

	ErrCategoryAlreadyExists = errors.New(
		"category already exists under the selected parent",
	)

	ErrInvalidProductCodePrefix = errors.New(
		"product-code prefix must use the format AAA-BBB with uppercase letters or digits",
	)

	ErrProductCodePrefixInUse = errors.New(
		"product-code prefix is already assigned to another category",
	)
)
