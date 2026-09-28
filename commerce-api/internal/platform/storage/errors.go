package storage

import "errors"

var (
	ErrDisabled = errors.New(
		"object storage is disabled",
	)

	ErrNotFound = errors.New(
		"object not found",
	)

	ErrInvalidKey = errors.New(
		"invalid object key",
	)
)
