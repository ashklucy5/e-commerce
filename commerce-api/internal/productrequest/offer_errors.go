package productrequest

import "errors"

var (
	ErrOfferNotFound = errors.New(
		"product sourcing offer not found",
	)

	ErrOfferConflict = errors.New(
		"product sourcing offer conflicts with current request state",
	)

	ErrOfferNotActionable = errors.New(
		"product sourcing offer is not actionable",
	)

	ErrOfferExpired = errors.New(
		"product sourcing offer has expired",
	)

	ErrOfferAlreadyFinalized = errors.New(
		"product sourcing request already has a finalized confirmation",
	)

	ErrConfirmationNotFound = errors.New(
		"product sourcing confirmation not found",
	)
)
