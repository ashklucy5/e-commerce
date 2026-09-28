package crm

import "encoding/json"

type CreateCaseRequest struct {
	CaseType string `json:"type"`
	Subject  string `json:"subject"`
	Message  string `json:"message"`

	Attachments json.RawMessage `json:"attachments,omitempty"`

	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	OrderID   string `json:"order_id"`

	RequestedQuantity int `json:"requested_quantity"`
}

type AddMessageRequest struct {
	Message     string          `json:"message"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
}

type ListResult struct {
	Items  []Case `json:"items"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type MessageListResult struct {
	Items  []Message `json:"items"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}
