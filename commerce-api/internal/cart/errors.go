package cart

import "errors"

var (
	ErrInvalidCartKey = errors.New(
		"invalid cart key",
	)

	ErrCartNotFound = errors.New(
		"cart not found",
	)

	ErrCartInactive = errors.New(
		"cart is no longer active",
	)

	ErrCartExpired = errors.New(
		"cart has expired",
	)

	ErrInvalidVariantID = errors.New(
		"invalid variant id",
	)

	ErrVariantUnavailable = errors.New(
		"variant is unavailable",
	)

	ErrInvalidItemID = errors.New(
		"invalid cart item id",
	)

	ErrItemNotFound = errors.New(
		"cart item not found",
	)

	ErrInvalidQuantity = errors.New(
		"quantity must be greater than zero",
	)

	ErrBelowMinimumOrderQuantity = errors.New(
		"quantity is below minimum order quantity",
	)

	ErrInvalidOrderIncrement = errors.New(
		"quantity does not match the required order increment",
	)

	ErrInsufficientStock = errors.New(
		"insufficient stock",
	)

	ErrCurrencyMismatch = errors.New(
		"variant currency does not match cart currency",
	)

	ErrMoneyOverflow = errors.New(
		"cart total exceeds supported amount",
	)
)
