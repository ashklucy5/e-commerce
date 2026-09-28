package support

import "errors"

var (
	ErrNotSupportActor = errors.New(
		"staff account is not a support actor",
	)

	ErrSupportActorDisabled = errors.New(
		"support actor is disabled",
	)

	ErrInvalidPresence = errors.New(
		"invalid support presence",
	)

	ErrAssignmentQueueUnavailable = errors.New(
		"support assignment queue unavailable",
	)

	ErrCaseNotFound = errors.New(
		"support case not found",
	)

	ErrCaseAlreadyClaimed = errors.New(
		"support case is already claimed",
	)

	ErrCaseNotOwned = errors.New(
		"support case is not owned by this actor",
	)

	ErrCaseClosed = errors.New(
		"support case is closed",
	)

	ErrCaseNotResolved = errors.New(
		"support case must be resolved before it can be closed",
	)

	ErrActorCapacity = errors.New(
		"support actor has reached active case capacity",
	)

	ErrInvalidMessage = errors.New(
		"invalid support message",
	)

	ErrInvalidVisibility = errors.New(
		"invalid support message visibility",
	)

	ErrInvalidQueue = errors.New(
		"invalid support queue",
	)
)
