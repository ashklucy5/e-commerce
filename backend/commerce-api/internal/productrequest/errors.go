package productrequest

import "errors"

var (
	ErrInvalidRequest     = errors.New("invalid product request")
	ErrNotFound           = errors.New("product request not found")
	ErrConversationClosed = errors.New(
		"product request conversation is closed",
	)
)
