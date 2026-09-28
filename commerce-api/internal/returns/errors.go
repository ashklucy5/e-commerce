package returns

import "errors"

var (
	ErrInvalidInput = errors.New(
		"invalid return request",
	)

	ErrReturnNotFound = errors.New(
		"return not found",
	)

	ErrOrderNotFound = errors.New(
		"order not found",
	)

	ErrReturnNotAllowed = errors.New(
		"order is not eligible for return",
	)

	ErrReturnItemNotFound = errors.New(
		"return item does not belong to the order",
	)

	ErrDuplicateReturnItem = errors.New(
		"duplicate return item",
	)

	ErrReturnQuantityExceeded = errors.New(
		"return quantity exceeds purchased quantity",
	)

	ErrInvalidReturnTransition = errors.New(
		"return transition is not allowed",
	)

	ErrRejectionReasonRequired = errors.New(
		"return rejection reason is required",
	)

	ErrInvalidReceivedQuantity = errors.New(
		"invalid received quantity",
	)

	ErrInvalidInspection = errors.New(
		"invalid return inspection",
	)

	ErrInventoryRestockFailed = errors.New(
		"unable to restock returned inventory",
	)
)
