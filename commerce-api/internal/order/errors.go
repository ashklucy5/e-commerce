package order

import "errors"

var (
	ErrInvalidCheckoutKey = errors.New(
		"invalid checkout key",
	)

	ErrInvalidOrderID = errors.New(
		"invalid order id",
	)

	ErrCheckoutNotFound = errors.New(
		"checkout not found",
	)

	ErrCheckoutNotActive = errors.New(
		"checkout is not active",
	)

	ErrCheckoutExpired = errors.New(
		"checkout has expired",
	)

	ErrCartNotActive = errors.New(
		"cart is not active",
	)

	ErrCustomerDetailsRequired = errors.New(
		"customer details are required",
	)

	ErrShippingDetailsRequired = errors.New(
		"shipping details are required",
	)

	ErrInvalidPaymentMethod = errors.New(
		"invalid payment method",
	)

	ErrEmptyCheckout = errors.New(
		"checkout contains no items",
	)

	ErrItemUnavailable = errors.New(
		"checkout item is no longer available",
	)

	ErrBelowMinimumOrderQuantity = errors.New(
		"quantity is below the current minimum order quantity",
	)

	ErrCheckoutChanged = errors.New(
		"checkout pricing or product terms have changed",
	)

	ErrInsufficientStock = errors.New(
		"insufficient stock",
	)

	ErrInventoryReservationMismatch = errors.New(
		"inventory reservation does not match order items",
	)

	ErrOrderNotFound = errors.New(
		"order not found",
	)

	ErrOrderNotPendingPayment = errors.New(
		"order is not pending payment",
	)

	ErrPaymentWindowExpired = errors.New(
		"payment window has expired",
	)

	ErrCancellationReasonRequired = errors.New(
		"cancellation reason is required",
	)

	ErrCancellationNotAllowed = errors.New(
		"order can no longer be cancelled",
	)

	ErrRefundRequired = errors.New(
		"paid online orders must be refunded before cancellation",
	)

	ErrMoneyOverflow = errors.New(
		"money calculation overflow",
	)

	ErrInvalidFulfillmentStatus = errors.New(
		"invalid fulfillment status",
	)

	ErrFulfillmentTransitionNotAllowed = errors.New(
		"fulfillment transition is not allowed",
	)

	ErrCourierNameRequired = errors.New(
		"courier name is required when shipping an order",
	)

	ErrInvalidFulfillmentDetails = errors.New(
		"invalid fulfillment details",
	)

	ErrPaymentNotReadyForFulfillment = errors.New(
		"order payment is not ready for fulfillment",
	)

	ErrShipmentNotFound = errors.New(
		"shipment not found for order",
	)
)
