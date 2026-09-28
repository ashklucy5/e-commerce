package refund

import "time"

const (
	StatusRequested  = "requested"
	StatusApproved   = "approved"
	StatusProcessing = "processing"
	StatusSucceeded  = "succeeded"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

const (
	SourceCancellation = "cancellation"
	SourceReturn       = "return"
	SourceManual       = "manual"
)

const (
	ProviderBKash        = "bkash"
	ProviderNagad        = "nagad"
	ProviderRocket       = "rocket"
	ProviderBankTransfer = "bank_transfer"
	ProviderManual       = "manual"
)

type Refund struct {
	ID           string `json:"id"`
	RefundNumber string `json:"refund_number"`

	OrderID  string `json:"order_id"`
	ReturnID string `json:"return_id,omitempty"`

	PaymentID string `json:"payment_id,omitempty"`

	SourceType string `json:"source_type"`
	Status     string `json:"status"`

	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`

	Provider         string `json:"provider,omitempty"`
	ProviderRefundID string `json:"provider_refund_id,omitempty"`

	Reason string `json:"reason"`

	RequestedBy string `json:"requested_by,omitempty"`
	ApprovedBy  string `json:"approved_by,omitempty"`

	FailureCode    string `json:"failure_code,omitempty"`
	FailureMessage string `json:"failure_message,omitempty"`

	RequestedAt  time.Time  `json:"requested_at"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	ProcessingAt *time.Time `json:"processing_at,omitempty"`
	SucceededAt  *time.Time `json:"succeeded_at,omitempty"`
	FailedAt     *time.Time `json:"failed_at,omitempty"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
