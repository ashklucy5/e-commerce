package review

import "errors"

var (
	ErrInvalidRequest = errors.New(
		"invalid review request",
	)

	ErrOrderItemNotFound = errors.New(
		"review order item not found",
	)

	ErrOrderNotEligible = errors.New(
		"order is not eligible for review",
	)

	ErrAlreadyExists = errors.New(
		"active review already exists for order item",
	)

	ErrReviewNotFound = errors.New(
		"review not found",
	)

	ErrProductNotFound = errors.New(
		"review product not found",
	)
)
