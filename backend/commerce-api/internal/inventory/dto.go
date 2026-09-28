package inventory

import "time"

type AdjustRequest struct {
	QuantityDelta int    `json:"quantity_delta"`
	Reason        string `json:"reason"`
	Note          string `json:"note,omitempty"`
	ActorType     string `json:"actor_type,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}

type AdjustReferenceInput struct {
	VariantID     string
	QuantityDelta int

	ReferenceType string
	ReferenceID   string

	Reason string
	Note   string

	ActorType string
	ActorID   string
}

type UpdateRequest struct {
	ReorderLevel *int `json:"reorder_level"`
}

type ReserveItem struct {
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
}

type ReserveReferenceInput struct {
	ReferenceType string        `json:"reference_type"`
	ReferenceID   string        `json:"reference_id"`
	Items         []ReserveItem `json:"items"`
	ExpiresAt     *time.Time    `json:"expires_at,omitempty"`
	ActorType     string        `json:"actor_type,omitempty"`
	ActorID       string        `json:"actor_id,omitempty"`
}

type ReferenceActionInput struct {
	ReferenceType string `json:"reference_type"`
	ReferenceID   string `json:"reference_id"`
	Reason        string `json:"reason,omitempty"`
	Note          string `json:"note,omitempty"`
	ActorType     string `json:"actor_type,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
}
