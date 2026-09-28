package inventory

import "errors"

var (
	ErrNotFound = errors.New(
		"inventory variant not found",
	)

	ErrInvalidInput = errors.New(
		"invalid inventory input",
	)

	ErrInsufficientStock = errors.New(
		"insufficient available inventory",
	)

	ErrReservationClosed = errors.New(
		"inventory reservation is already closed",
	)

	ErrInventoryInvariant = errors.New(
		"inventory balance invariant violated",
	)
)
