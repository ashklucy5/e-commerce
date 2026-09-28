package payment

import "time"

type Event struct {
	ID string `json:"id"`

	OrderID   string `json:"order_id"`
	PaymentID string `json:"payment_id,omitempty"`

	Provider string `json:"provider"`

	ProviderEventID string `json:"provider_event_id"`
	EventType       string `json:"event_type"`

	ProviderPaymentID string `json:"provider_payment_id"`

	ProviderTransactionID string `json:"provider_transaction_id,omitempty"`

	PayloadSHA256 string `json:"payload_sha256"`

	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`

	ReceivedAt  time.Time  `json:"received_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}
