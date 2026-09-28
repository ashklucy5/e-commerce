package notification

import "errors"

var (
	ErrInvalidInput = errors.New(
		"invalid notification input",
	)

	ErrOutboxNotFound = errors.New(
		"notification outbox item not found",
	)

	ErrChannelNotConfigured = errors.New(
		"notification channel is not configured",
	)

	ErrChannelUnavailable = errors.New(
		"notification channel is unavailable",
	)

	ErrTemplateNotFound = errors.New(
		"notification template not found",
	)

	ErrTemplateInvalid = errors.New(
		"notification template is invalid",
	)
)
