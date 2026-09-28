package productrequest

import (
	"encoding/json"
	"time"
)

type AdminCustomer struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email,omitempty"`
}

type AdminActor struct {
	ID          string `json:"id"`
	ActorCode   string `json:"actor_code"`
	ActorType   string `json:"actor_type"`
	DisplayName string `json:"display_name"`
}

type AdminAssignment struct {
	QueueCode string `json:"queue_code"`
	QueueName string `json:"queue_name"`

	SupportActor *AdminActor `json:"support_actor,omitempty"`
}

type AdminRequest struct {
	ID            string `json:"id"`
	RequestNumber string `json:"request_number"`

	CaseID     string `json:"case_id"`
	CaseNumber string `json:"case_number"`

	RequestedProductName string `json:"requested_product_name"`
	Description          string `json:"description"`
	RequestedQuantity    int    `json:"requested_quantity"`

	CustomerRequirements json.RawMessage `json:"customer_requirements,omitempty"`
	ExternalURL          string          `json:"external_url,omitempty"`
	Attachments          json.RawMessage `json:"attachments,omitempty"`

	Status       string `json:"status"`
	StatusReason string `json:"status_reason,omitempty"`

	CRMStatus   string `json:"crm_status"`
	CRMPriority string `json:"crm_priority"`

	Customer   AdminCustomer    `json:"customer"`
	ReviewedBy *AdminActor      `json:"reviewed_by,omitempty"`
	Assignment *AdminAssignment `json:"assignment,omitempty"`

	LastMessageAt         time.Time  `json:"last_message_at"`
	LastCustomerMessageAt *time.Time `json:"last_customer_message_at,omitempty"`
	LastSupportMessageAt  *time.Time `json:"last_support_message_at,omitempty"`
	ReviewedAt            *time.Time `json:"reviewed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminMessage struct {
	ID         string `json:"id"`
	AuthorType string `json:"author_type"`
	Visibility string `json:"visibility"`
	Body       string `json:"body"`

	SupportActor *AdminActor `json:"support_actor,omitempty"`

	CreatedAt   time.Time       `json:"created_at"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
}

type AdminListResult struct {
	Items  []AdminRequest `json:"items"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type AdminMessageListResult struct {
	Items  []AdminMessage `json:"items"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}
