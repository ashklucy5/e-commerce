package refund

import "time"

const (
	EventRequested  = "refund_requested"
	EventApproved   = "refund_approved"
	EventProcessing = "refund_processing"
	EventSucceeded  = "refund_succeeded"
	EventFailed     = "refund_failed"
	EventCancelled  = "refund_cancelled"
)

type Event struct {
	ID       string `json:"id"`
	RefundID string `json:"refund_id"`

	EventType string `json:"event_type"`

	FromStatus string `json:"from_status,omitempty"`
	ToStatus   string `json:"to_status,omitempty"`

	Message string `json:"message,omitempty"`

	ActorType string `json:"actor_type,omitempty"`
	ActorID   string `json:"actor_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

type eventInsert struct {
	EventType string

	FromStatus string
	ToStatus   string

	Message string

	ActorType string
	ActorID   string
}
