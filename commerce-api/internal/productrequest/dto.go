package productrequest

import "encoding/json"

type CreateRequest struct {
	RequestedProductName string `json:"requested_product_name"`
	Description          string `json:"description"`
	RequestedQuantity    int    `json:"requested_quantity"`

	CustomerRequirements json.RawMessage `json:"customer_requirements,omitempty"`
	ExternalURL          string          `json:"external_url,omitempty"`
	Attachments          json.RawMessage `json:"attachments,omitempty"`
}

type AddMessageRequest struct {
	Message     string          `json:"message"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
}

type ListResult struct {
	Items  []Request `json:"items"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

type MessageListResult struct {
	Items  []Message `json:"items"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}
