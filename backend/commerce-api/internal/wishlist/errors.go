package wishlist

import "errors"

var (
	ErrAuthenticationRequired = errors.New(
		"customer authentication is required",
	)

	ErrInvalidProductID = errors.New(
		"invalid product id",
	)

	ErrProductNotFound = errors.New(
		"product not found",
	)
)
