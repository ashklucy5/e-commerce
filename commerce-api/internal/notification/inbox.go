package notification

import (
	"encoding/json"
	"time"
)

const (
	StaffCategoryOrder     = "order"
	StaffCategoryInventory = "inventory"
	StaffCategorySourcing  = "sourcing"
	StaffCategorySupport   = "support"
	StaffCategoryReturn    = "return"
	StaffCategoryDelivery  = "delivery"
	StaffCategorySecurity  = "security"
	StaffCategorySystem    = "system"
)

const (
	StaffPriorityInfo      = "info"
	StaffPriorityAttention = "attention"
	StaffPriorityCritical  = "critical"
)

const (
	StaffEventOrderPlaced          = "order.placed"
	StaffEventProductRequestNew    = "sourcing.request.created"
	StaffEventProductRequestReply  = "sourcing.customer_reply"
	StaffEventSupportCaseNew       = "support.case.created"
	StaffEventSupportCustomerReply = "support.customer_reply"
	StaffEventInventoryLowStock    = "inventory.low_stock"
)

type InboxItem struct {
	ID string `json:"id"`

	Category  string `json:"category"`
	EventType string `json:"event_type"`

	Title   string `json:"title"`
	Message string `json:"message"`

	ActionURL string `json:"action_url,omitempty"`

	OrderID string `json:"order_id,omitempty"`
	CaseID  string `json:"case_id,omitempty"`

	EntityType string `json:"entity_type,omitempty"`
	EntityID   string `json:"entity_id,omitempty"`

	Priority string `json:"priority,omitempty"`

	Metadata json.RawMessage `json:"metadata"`

	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type InboxListResult struct {
	Items []InboxItem `json:"items"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type InboxSummary struct {
	UnreadCount int64 `json:"unread_count"`
}

type InboxListOptions struct {
	Limit int

	Offset int

	UnreadOnly bool

	Category string
}

type CustomerInboxRequest struct {
	DedupeKey string

	CustomerID string

	Category  string
	EventType string

	Title   string
	Message string

	ActionURL string

	OrderID string
	CaseID  string

	Metadata map[string]any
}

type StaffEventRequest struct {
	DedupeKey string

	Category  string
	EventType string
	Priority  string

	Title   string
	Message string

	ActionURL string

	EntityType string
	EntityID   string

	RequiredPermission string

	Metadata map[string]any
}
