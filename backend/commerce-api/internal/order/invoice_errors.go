package order

import (
	"errors"
	"fmt"
)

var ErrInvoiceNotFound = fmt.Errorf(
	"%w: invoice not found",
	ErrOrderNotFound,
)

var ErrInvalidInvoiceResendKey = errors.New(
	"invalid invoice resend idempotency key",
)
