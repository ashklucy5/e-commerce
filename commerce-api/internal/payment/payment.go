package payment

import "time"

type Payment struct {
	ID      string `json:"id"`
	OrderID string `json:"order_id"`

	Provider string `json:"provider"`
	Status   string `json:"status"`

	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`

	ProviderPaymentID string `json:"provider_payment_id"`

	ProviderTransactionID string `json:"provider_transaction_id,omitempty"`

	PaidAt   *time.Time `json:"paid_at,omitempty"`
	FailedAt *time.Time `json:"failed_at,omitempty"`

	FailureCode    string `json:"failure_code,omitempty"`
	FailureMessage string `json:"failure_message,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
