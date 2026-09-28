package productrequest

import "errors"

var (
	ErrInvalidStatusTransition = errors.New(
		"invalid product request status transition",
	)

	ErrAdminActorUnavailable = errors.New(
		"admin support actor is unavailable",
	)
)
