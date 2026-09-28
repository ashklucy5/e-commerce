package productrequest

import (
	"encoding/json"
	"time"
)

type SourcingOffer struct {
	ID        string `json:"id"`
	RequestID string `json:"request_id"`
	Status    string `json:"status"`

	ProductName string `json:"product_name"`
	Description string `json:"description"`

	Attachments           json.RawMessage `json:"attachments,omitempty"`
	OfferedSpecifications json.RawMessage `json:"offered_specifications,omitempty"`

	UnitPrice     int64  `json:"unit_price"`
	ShippingPrice int64  `json:"shipping_price"`
	Currency      string `json:"currency"`

	QuotedQuantity       *int `json:"quoted_quantity,omitempty"`
	MinimumOrderQuantity *int `json:"minimum_order_quantity,omitempty"`

	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	SentAt              *time.Time `json:"sent_at,omitempty"`
	CustomerRespondedAt *time.Time `json:"customer_responded_at,omitempty"`
	FinalizedAt         *time.Time `json:"finalized_at,omitempty"`

	CreatedBy *AdminActor `json:"created_by,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SourcingConfirmation struct {
	ID        string `json:"id"`
	RequestID string `json:"request_id"`
	OfferID   string `json:"offer_id"`

	Quantity               int             `json:"quantity"`
	MinimumOrderQuantity   int             `json:"minimum_order_quantity"`
	AcceptedProductName    string          `json:"accepted_product_name"`
	AcceptedSpecifications json.RawMessage `json:"accepted_specifications,omitempty"`

	UnitPriceSnapshot     int64  `json:"unit_price_snapshot"`
	ShippingPriceSnapshot int64  `json:"shipping_price_snapshot"`
	Currency              string `json:"currency"`
	TotalAmount           int64  `json:"total_amount"`
	Status                string `json:"status"`

	FinalizedBy *AdminActor `json:"finalized_by,omitempty"`

	CreatedProductID *string `json:"created_product_id,omitempty"`
	CreatedVariantID *string `json:"created_variant_id,omitempty"`
	CreatedOrderID   *string `json:"created_order_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminCreateOfferRequest struct {
	ProductName string `json:"product_name"`
	Description string `json:"description"`

	Attachments           json.RawMessage `json:"attachments"`
	OfferedSpecifications json.RawMessage `json:"offered_specifications"`

	UnitPrice     int64  `json:"unit_price"`
	ShippingPrice int64  `json:"shipping_price"`
	Currency      string `json:"currency"`

	QuotedQuantity       *int       `json:"quoted_quantity"`
	MinimumOrderQuantity *int       `json:"minimum_order_quantity"`
	ExpiresAt            *time.Time `json:"expires_at"`
}

type AdminUpdateOfferRequest struct {
	ProductName *string `json:"product_name"`
	Description *string `json:"description"`

	Attachments           *json.RawMessage `json:"attachments"`
	OfferedSpecifications *json.RawMessage `json:"offered_specifications"`

	UnitPrice     *int64  `json:"unit_price"`
	ShippingPrice *int64  `json:"shipping_price"`
	Currency      *string `json:"currency"`

	QuotedQuantity       *int       `json:"quoted_quantity"`
	MinimumOrderQuantity *int       `json:"minimum_order_quantity"`
	ExpiresAt            *time.Time `json:"expires_at"`
}

type offerMutation struct {
	ProductName string
	Description string

	Attachments           json.RawMessage
	OfferedSpecifications json.RawMessage

	UnitPrice     int64
	ShippingPrice int64
	Currency      string

	QuotedQuantity       *int
	MinimumOrderQuantity *int
	ExpiresAt            *time.Time
}
