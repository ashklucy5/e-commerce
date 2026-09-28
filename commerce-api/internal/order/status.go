package order

const (
	OrderTypeStandard = "standard"
	OrderTypeSourcing = "sourcing"
)

const (
	StatusPendingPayment      = "pending_payment"
	StatusAwaitingProcurement = "awaiting_procurement"
	StatusConfirmed           = "confirmed"
	StatusProcessing          = "processing"
	StatusShipped             = "shipped"
	StatusDelivered           = "delivered"
	StatusCompleted           = "completed"
	StatusPaymentExpired      = "payment_expired"
	StatusCancelled           = "cancelled"
)

const (
	PaymentStatusPending      = "pending"
	PaymentStatusPaid         = "paid"
	PaymentStatusFailed       = "failed"
	PaymentStatusExpired      = "expired"
	PaymentStatusCODPending   = "cod_pending"
	PaymentStatusCODCollected = "cod_collected"
	PaymentStatusRefunded     = "refunded"
)

const (
	PaymentMethodCOD          = "cod"
	PaymentMethodBKash        = "bkash"
	PaymentMethodNagad        = "nagad"
	PaymentMethodRocket       = "rocket"
	PaymentMethodBankTransfer = "bank_transfer"
)

const inventoryReferenceType = "order"
