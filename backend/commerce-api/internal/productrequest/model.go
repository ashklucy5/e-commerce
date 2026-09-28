package productrequest

import (
	"encoding/json"
	"time"
)

type Request struct {
	ID            string `json:"id"`
	RequestNumber string `json:"request_number"`

	RequestedProductName string `json:"requested_product_name"`
	Description          string `json:"description"`
	RequestedQuantity    int    `json:"requested_quantity"`

	CustomerRequirements json.RawMessage `json:"customer_requirements,omitempty"`
	ExternalURL          string          `json:"external_url,omitempty"`
	Attachments          json.RawMessage `json:"attachments,omitempty"`

	Status       string `json:"status"`
	StatusReason string `json:"status_reason,omitempty"`

	CanMessage    bool       `json:"can_message"`
	LastMessageAt time.Time  `json:"last_message_at"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID string `json:"id"`

	AuthorType string `json:"author_type"`
	Visibility string `json:"visibility"`
	Body       string `json:"body"`

	SupportActorCode string `json:"support_actor_code,omitempty"`
	SupportActorType string `json:"support_actor_type,omitempty"`
	SupportActorName string `json:"support_actor_name,omitempty"`

	CreatedAt   time.Time       `json:"created_at"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
}
