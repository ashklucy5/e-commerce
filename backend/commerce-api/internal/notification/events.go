package notification

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

const (
	CategoryOrder          = "order"
	CategoryPayment        = "payment"
	CategoryDelivery       = "delivery"
	CategorySourcing       = "sourcing"
	CategorySupport        = "support"
	CategoryPromotion      = "promotion"
	CategoryRecommendation = "recommendation"
	CategorySecurity       = "security"
	CategorySystem         = "system"
)

const (
	ChannelSMS       = "sms"
	ChannelEmail     = "email"
	ChannelPush      = "push"
	ChannelMessenger = "messenger"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSent       = "sent"
	StatusSkipped    = "skipped"
	StatusDead       = "dead"
)

const (
	EventOrderPlaced = "order.placed"

	EventOrderProcessing = "order.processing"

	EventPaymentSucceeded = "payment.succeeded"

	EventDeliveryUpdated = "delivery.updated"

	EventDeliverySent = "delivery.dispatched"

	EventDeliveryComplete = "delivery.delivered"

	EventSupportReply = "support.reply"

	EventInvoiceIssued = "invoice.issued"

	EventInvoiceResent = "invoice.resent"
)

const maxOutboxPayloadBytes = 64 << 10

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`,
)

type OutboxItem struct {
	ID string `json:"id"`

	DedupeKey string `json:"dedupe_key"`

	Category  string `json:"category"`
	EventType string `json:"event_type"`
	Channel   string `json:"channel"`

	CustomerID string `json:"customer_id,omitempty"`
	OrderID    string `json:"order_id,omitempty"`
	CaseID     string `json:"case_id,omitempty"`

	Recipient string `json:"recipient,omitempty"`

	TemplateKey string `json:"template_key"`

	Payload json.RawMessage `json:"payload"`

	Status string `json:"status"`

	AttemptCount int `json:"attempt_count"`
	MaxAttempts  int `json:"max_attempts"`

	AvailableAt time.Time `json:"available_at"`

	LockedAt *time.Time `json:"locked_at,omitempty"`

	ProcessedAt *time.Time `json:"processed_at,omitempty"`

	ProviderMessageID string `json:"provider_message_id,omitempty"`

	LastError string `json:"last_error,omitempty"`

	SkipReason string `json:"skip_reason,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

type EnqueueRequest struct {
	DedupeKey string

	Category string

	EventType string

	Channel string

	CustomerID string

	OrderID string

	CaseID string

	Recipient string

	TemplateKey string

	Payload map[string]any

	MaxAttempts int

	AvailableAt time.Time
}

func ChannelDedupeKey(
	base string,
	channel string,
) string {
	base =
		strings.TrimSpace(
			base,
		)

	channel =
		strings.ToLower(
			strings.TrimSpace(
				channel,
			),
		)

	if base == "" ||
		channel == "" {

		return ""
	}

	return base +
		":" +
		channel
}

func validCategory(
	value string,
) bool {
	switch value {
	case CategoryOrder,
		CategoryPayment,
		CategoryDelivery,
		CategorySourcing,
		CategorySupport,
		CategoryPromotion,
		CategoryRecommendation,
		CategorySecurity,
		CategorySystem:

		return true

	default:

		return false
	}
}

func validChannel(
	value string,
) bool {
	switch value {
	case ChannelSMS,
		ChannelEmail,
		ChannelPush,
		ChannelMessenger:

		return true

	default:

		return false
	}
}

func validOptionalUUID(
	value string,
) bool {
	return value == "" ||
		uuidPattern.MatchString(
			value,
		)
}
