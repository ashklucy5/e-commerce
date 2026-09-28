package productrequest

import "encoding/json"

type AdminListFilter struct {
	Status string
	Query  string
	Limit  int
	Offset int
}

type AdminAddMessageRequest struct {
	Message     string          `json:"message"`
	Visibility  string          `json:"visibility,omitempty"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
}

type AdminUpdateStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}
