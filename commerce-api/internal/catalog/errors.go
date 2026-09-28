package catalog

import "errors"

var (
	ErrProductNotFound = errors.New(
		"product not found",
	)

	ErrInvalidProductSlug = errors.New(
		"product slug is required",
	)
)
