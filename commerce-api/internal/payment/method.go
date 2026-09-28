package payment

const (
	// COD is a commerce payment method but not an external
	// payment provider. It therefore never enters the external
	// payment-initiation pipeline.
	PaymentMethodCOD = "cod"
)

const (
	ProviderBKash        = "bkash"
	ProviderNagad        = "nagad"
	ProviderRocket       = "rocket"
	ProviderBankTransfer = "bank_transfer"
)

const (
	StatusPending   = "pending"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusExpired   = "expired"
	StatusRefunded  = "refunded"
)

const (
	EventStatusReceived  = "received"
	EventStatusProcessed = "processed"
	EventStatusIgnored   = "ignored"
	EventStatusFailed    = "failed"
)
