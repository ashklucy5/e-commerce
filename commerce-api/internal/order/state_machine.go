package order

import (
	"strings"
	"time"
)

const (
	onlinePaymentWindow       = 15 * time.Minute
	bankTransferPaymentWindow = 24 * time.Hour
)

type initialPaymentState struct {
	OrderStatus   string
	PaymentStatus string
	PaymentDueAt  *time.Time
	ConfirmedAt   *time.Time
}

func buildInitialPaymentState(
	paymentMethod string,
	now time.Time,
) (initialPaymentState, error) {
	method :=
		strings.ToLower(
			strings.TrimSpace(
				paymentMethod,
			),
		)

	switch method {
	case PaymentMethodCOD:
		confirmedAt := now

		return initialPaymentState{
			OrderStatus:   StatusConfirmed,
			PaymentStatus: PaymentStatusCODPending,
			ConfirmedAt:   &confirmedAt,
		}, nil

	case PaymentMethodBKash,
		PaymentMethodNagad,
		PaymentMethodRocket:
		dueAt :=
			now.Add(
				onlinePaymentWindow,
			)

		return initialPaymentState{
			OrderStatus:   StatusPendingPayment,
			PaymentStatus: PaymentStatusPending,
			PaymentDueAt:  &dueAt,
		}, nil

	case PaymentMethodBankTransfer:
		dueAt :=
			now.Add(
				bankTransferPaymentWindow,
			)

		return initialPaymentState{
			OrderStatus:   StatusPendingPayment,
			PaymentStatus: PaymentStatusPending,
			PaymentDueAt:  &dueAt,
		}, nil

	default:
		return initialPaymentState{},
			ErrInvalidPaymentMethod
	}
}
