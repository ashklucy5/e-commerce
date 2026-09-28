package payment

import "time"

type ConfirmVerifiedPaymentInput struct {
	OrderID string

	Provider string

	ProviderEventID string
	EventType       string

	ProviderPaymentID string

	ProviderTransactionID string

	Amount   int64
	Currency string

	PaidAt time.Time

	PayloadSHA256 string
}

type ConfirmVerifiedPaymentResult struct {
	Payment Payment

	Processed bool
}
