package delivery

import "errors"

var (
	ErrInvalidInput                   = errors.New("invalid delivery request")
	ErrOrderNotFound                  = errors.New("order not found")
	ErrShipmentNotFound               = errors.New("shipment not found")
	ErrShipmentAlreadyExists          = errors.New("shipment already exists for order")
	ErrOrderNotReady                  = errors.New("order is not ready for delivery")
	ErrPaymentNotReady                = errors.New("order payment is not ready for delivery")
	ErrWarehouseNotReady              = errors.New("warehouse fulfillment is not ready for delivery")
	ErrMultipleWarehousesNotSupported = errors.New("multiple origin warehouses are not supported while one shipment per order is enabled")
	ErrShipmentNotReady               = errors.New("shipment is not ready for this delivery transition")
	ErrHandoffIncomplete              = errors.New("warehouse handoff is incomplete")
	ErrDeliveryModeMismatch           = errors.New("delivery mode does not support this operation")
	ErrCustomerAuthenticationRequired = errors.New("customer authentication is required")
	ErrCustomerOrderMismatch          = errors.New("order does not belong to the authenticated customer")

	ErrETAEstimatorUnavailable = errors.New(
		"delivery ETA estimator is not configured",
	)

	ErrDeliveryPricerUnavailable = errors.New(
		"delivery pricer is not configured",
	)

	ErrOTPSenderUnavailable = errors.New(
		"OTP sender is not configured",
	)

	ErrOTPChallengeNotFound = errors.New(
		"OTP challenge not found",
	)

	ErrOTPChallengeExpired = errors.New(
		"OTP challenge has expired",
	)

	ErrOTPChallengeLocked = errors.New(
		"OTP challenge is locked",
	)

	ErrOTPInvalidCode = errors.New(
		"invalid OTP code",
	)

	ErrOTPChallengeNotVerified = errors.New(
		"OTP challenge has not been verified",
	)

	ErrOTPChallengeConsumed = errors.New(
		"OTP challenge has already been consumed",
	)

	ErrProofNotFound = errors.New(
		"delivery proof not found",
	)
)
