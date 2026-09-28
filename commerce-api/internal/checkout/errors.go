package checkout

import "errors"

var (
	ErrInvalidCartKey = errors.New(
		"invalid cart key",
	)

	ErrInvalidCheckoutKey = errors.New(
		"invalid checkout key",
	)

	ErrCartNotFound = errors.New(
		"cart not found",
	)

	ErrCartInactive = errors.New(
		"cart is not active",
	)

	ErrCartExpired = errors.New(
		"cart has expired",
	)

	ErrEmptyCart = errors.New(
		"cart is empty",
	)

	ErrCartItemUnavailable = errors.New(
		"cart contains an unavailable item",
	)

	ErrBelowMinimumOrderQuantity = errors.New(
		"cart item is below minimum order quantity",
	)

	ErrInsufficientStock = errors.New(
		"insufficient stock",
	)

	ErrCurrencyMismatch = errors.New(
		"cart contains a currency mismatch",
	)

	ErrCheckoutNotFound = errors.New(
		"checkout not found",
	)

	ErrCheckoutNotActive = errors.New(
		"checkout is not active",
	)

	ErrInvalidCustomerDetails = errors.New(
		"invalid customer details",
	)

	ErrInvalidShippingDetails = errors.New(
		"invalid shipping details",
	)

	ErrInvalidPaymentMethod = errors.New(
		"invalid payment method",
	)

	ErrInvalidDeliveryMethod = errors.New(
		"invalid delivery method",
	)

	ErrMoneyOverflow = errors.New(
		"checkout amount exceeds supported range",
	)

	ErrInvalidPromotionCode = errors.New(
		"invalid promotion code",
	)

	ErrPromotionNotFound = errors.New(
		"promotion not found",
	)

	ErrPromotionNotApplicable = errors.New(
		"promotion is not applicable",
	)

	ErrPromotionUnavailable = errors.New(
		"promotion service is unavailable",
	)
)
