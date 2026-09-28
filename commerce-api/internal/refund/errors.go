package refund

import "errors"

var (
	ErrInvalidInput = errors.New(
		"invalid refund request",
	)

	ErrRefundNotFound = errors.New(
		"refund not found",
	)

	ErrReturnNotFound = errors.New(
		"return not found",
	)

	ErrOrderNotFound = errors.New(
		"refund order not found",
	)

	ErrPaymentNotFound = errors.New(
		"successful payment not found",
	)

	ErrReturnNotReady = errors.New(
		"return is not ready for refund",
	)

	ErrRefundNotAllowed = errors.New(
		"refund is not allowed",
	)

	ErrNothingToRefund = errors.New(
		"there is no refundable amount",
	)

	ErrRefundAmountExceeded = errors.New(
		"refund amount exceeds remaining refundable amount",
	)

	ErrInvalidRefundTransition = errors.New(
		"refund transition is not allowed",
	)

	ErrProviderRefundIDRequired = errors.New(
		"provider refund reference is required",
	)

	ErrFailureDetailsRequired = errors.New(
		"refund failure details are required",
	)
)
