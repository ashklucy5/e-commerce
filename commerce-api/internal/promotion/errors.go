package promotion

import "errors"

var (
	ErrInvalidSubtotal = errors.New(
		"subtotal amount must be nonnegative",
	)

	ErrInvalidCurrency = errors.New(
		"currency must be a three-letter code",
	)

	ErrInvalidCode = errors.New(
		"promotion code is invalid",
	)

	ErrPromotionNotFound = errors.New(
		"promotion not found",
	)

	ErrPromotionNotApplicable = errors.New(
		"promotion is not applicable",
	)

	ErrInvalidPromotion = errors.New(
		"promotion configuration is invalid",
	)
)
