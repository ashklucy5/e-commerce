package catalog

import "errors"

var (
	ErrInvalidCategoryFilter = errors.New(
		"invalid product category filter",
	)

	ErrInvalidPriceFilter = errors.New(
		"invalid product price filter",
	)

	ErrInvalidPriceRange = errors.New(
		"invalid product price range",
	)

	ErrInvalidStockFilter = errors.New(
		"invalid product stock filter",
	)

	ErrInvalidProductSort = errors.New(
		"invalid product sort",
	)
)
