package payment

import "errors"

var (
	ErrInvalidInput = errors.New(
		"invalid payment input",
	)

	ErrUnsupportedProvider = errors.New(
		"unsupported payment provider",
	)

	ErrOrderNotFound = errors.New(
		"payment order not found",
	)

	ErrPaymentNotFound = errors.New(
		"payment not found",
	)

	ErrPaymentMethodMismatch = errors.New(
		"payment provider does not match order payment method",
	)

	ErrPaymentAmountMismatch = errors.New(
		"verified payment amount does not match order total",
	)

	ErrPaymentCurrencyMismatch = errors.New(
		"verified payment currency does not match order currency",
	)

	ErrPaymentReferenceConflict = errors.New(
		"payment provider reference conflicts with an existing payment",
	)

	ErrReconciliationRequired = errors.New(
		"verified payment requires reconciliation before the order can continue",
	)

	// Payment initiation errors.

	ErrPaymentInitiationNotRequired = errors.New(
		"payment initiation not required",
	)

	ErrPaymentOrderNotPending = errors.New(
		"order is not awaiting payment",
	)

	ErrPaymentWindowExpired = errors.New(
		"payment window expired",
	)

	ErrPaymentMethodUnavailable = errors.New(
		"payment method unavailable",
	)

	ErrPaymentProviderRejected = errors.New(
		"payment provider rejected initiation",
	)

	ErrPaymentProviderOutcomeUnknown = errors.New(
		"payment provider initiation outcome unknown",
	)
)
