package admin

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	platformpagination "project.local/commerce-api/internal/platform/pagination"
)

const adminOperationalReadTimeout = 5 * time.Second

var (
	ErrAdminOrderNotFound   = errors.New("admin order not found")
	ErrAdminReturnNotFound  = errors.New("admin return not found")
	ErrAdminPaymentNotFound = errors.New("admin payment not found")
)

type adminReadQuerier interface {
	Query(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgx.Rows, error)

	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
}

type OrderReadFilter struct {
	Status        string
	PaymentStatus string
	Query         string
}

type ReturnReadFilter struct {
	Status string
	Query  string
}

type PaymentReadFilter struct {
	Status   string
	Provider string
	Query    string
}

type OrderListItem struct {
	ID          string `json:"id"`
	OrderNumber string `json:"order_number"`

	CustomerID    string `json:"customer_id,omitempty"`
	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`

	Status        string `json:"status"`
	PaymentStatus string `json:"payment_status"`
	PaymentMethod string `json:"payment_method"`

	Currency    string `json:"currency"`
	TotalAmount int64  `json:"total_amount"`

	DeliveryMethod string `json:"delivery_method"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrderItemRead struct {
	ID        string `json:"id"`
	VariantID string `json:"variant_id"`

	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	Quantity             int `json:"quantity"`
	MinimumOrderQuantity int `json:"minimum_order_quantity"`

	UnitPriceAmount int64  `json:"unit_price_amount"`
	LineTotalAmount int64  `json:"line_total_amount"`
	Currency        string `json:"currency"`
}

type PaymentSummary struct {
	ID string `json:"id"`

	Provider string `json:"provider"`
	Status   string `json:"status"`

	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`

	ProviderPaymentID     string `json:"provider_payment_id"`
	ProviderTransactionID string `json:"provider_transaction_id,omitempty"`

	PaidAt   *time.Time `json:"paid_at,omitempty"`
	FailedAt *time.Time `json:"failed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

type ReturnSummary struct {
	ID           string `json:"id"`
	ReturnNumber string `json:"return_number"`
	Status       string `json:"status"`

	RequestedAt time.Time `json:"requested_at"`
}

type RefundSummary struct {
	ID           string `json:"id"`
	RefundNumber string `json:"refund_number"`

	ReturnID  string `json:"return_id,omitempty"`
	PaymentID string `json:"payment_id,omitempty"`

	SourceType string `json:"source_type"`
	Status     string `json:"status"`

	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`

	Provider         string `json:"provider,omitempty"`
	ProviderRefundID string `json:"provider_refund_id,omitempty"`

	RequestedAt time.Time  `json:"requested_at"`
	SucceededAt *time.Time `json:"succeeded_at,omitempty"`
}

type OrderDetail struct {
	OrderListItem

	OrderType  string `json:"order_type"`
	CheckoutID string `json:"checkout_id"`
	CartID     string `json:"cart_id"`

	CustomerEmail string `json:"customer_email,omitempty"`

	SubtotalAmount int64 `json:"subtotal_amount"`
	DiscountAmount int64 `json:"discount_amount"`

	PromotionID   string `json:"promotion_id,omitempty"`
	PromotionCode string `json:"promotion_code,omitempty"`

	ShippingAmount int64 `json:"shipping_amount"`

	ShippingAddressLine1 string `json:"shipping_address_line1"`
	ShippingAddressLine2 string `json:"shipping_address_line2,omitempty"`
	ShippingCity         string `json:"shipping_city"`
	ShippingArea         string `json:"shipping_area"`
	ShippingPostalCode   string `json:"shipping_postal_code,omitempty"`

	PaymentDueAt *time.Time `json:"payment_due_at,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	ConfirmedAt  *time.Time `json:"confirmed_at,omitempty"`

	ProcessingAt *time.Time `json:"processing_at,omitempty"`
	ShippedAt    *time.Time `json:"shipped_at,omitempty"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty"`

	CancellationReason string `json:"cancellation_reason,omitempty"`
	CancelledBy        string `json:"cancelled_by,omitempty"`

	Items    []OrderItemRead  `json:"items"`
	Payments []PaymentSummary `json:"payments"`
	Returns  []ReturnSummary  `json:"returns"`
	Refunds  []RefundSummary  `json:"refunds"`
}

type ReturnListItem struct {
	ID           string `json:"id"`
	ReturnNumber string `json:"return_number"`

	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`

	Status string `json:"status"`

	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`

	RequestedAt time.Time `json:"requested_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ReturnItemRead struct {
	ID          string `json:"id"`
	OrderItemID string `json:"order_item_id"`
	VariantID   string `json:"variant_id"`

	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	Quantity int `json:"quantity"`

	ReasonCode string `json:"reason_code"`
	ReasonNote string `json:"reason_note,omitempty"`

	ReceivedQuantity int `json:"received_quantity"`
	RestockQuantity  int `json:"restock_quantity"`

	InspectionStatus string `json:"inspection_status"`
	InspectionNote   string `json:"inspection_note,omitempty"`

	UnitPriceAmount int64  `json:"unit_price_amount"`
	Currency        string `json:"currency"`
}

type ReturnDetail struct {
	ReturnListItem

	CustomerNote string `json:"customer_note,omitempty"`

	RequestedBy string `json:"requested_by,omitempty"`
	ApprovedBy  string `json:"approved_by,omitempty"`
	RejectedBy  string `json:"rejected_by,omitempty"`

	RejectionReason string `json:"rejection_reason,omitempty"`

	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	RejectedAt  *time.Time `json:"rejected_at,omitempty"`
	ReceivedAt  *time.Time `json:"received_at,omitempty"`
	InspectedAt *time.Time `json:"inspected_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`

	Items   []ReturnItemRead `json:"items"`
	Refunds []RefundSummary  `json:"refunds"`
}

type PaymentListItem struct {
	ID string `json:"id"`

	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`

	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`

	Provider string `json:"provider"`
	Status   string `json:"status"`

	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`

	ProviderPaymentID     string `json:"provider_payment_id"`
	ProviderTransactionID string `json:"provider_transaction_id,omitempty"`

	PaidAt   *time.Time `json:"paid_at,omitempty"`
	FailedAt *time.Time `json:"failed_at,omitempty"`

	FailureCode    string `json:"failure_code,omitempty"`
	FailureMessage string `json:"failure_message,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PaymentEventRead struct {
	ID string `json:"id"`

	ProviderEventID string `json:"provider_event_id"`
	EventType       string `json:"event_type"`

	ProviderPaymentID     string `json:"provider_payment_id"`
	ProviderTransactionID string `json:"provider_transaction_id,omitempty"`

	PayloadSHA256 string `json:"payload_sha256"`

	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`

	ReceivedAt  time.Time  `json:"received_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

type PaymentDetail struct {
	PaymentListItem

	Events  []PaymentEventRead `json:"events"`
	Refunds []RefundSummary    `json:"refunds"`
}

type OrderListResult struct {
	Items []OrderListItem
	Meta  platformpagination.Meta
}

type ReturnListResult struct {
	Items []ReturnListItem
	Meta  platformpagination.Meta
}

type PaymentListResult struct {
	Items []PaymentListItem
	Meta  platformpagination.Meta
}

func adminReadContext(
	ctx context.Context,
) (
	context.Context,
	context.CancelFunc,
) {
	return context.WithTimeout(
		ctx,
		adminOperationalReadTimeout,
	)
}

func adminTimePointer(
	value pgtype.Timestamptz,
) *time.Time {
	if !value.Valid {
		return nil
	}

	result := value.Time.UTC()

	return &result
}
