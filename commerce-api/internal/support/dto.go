package support

import "project.local/commerce-api/internal/staff"

type Actor struct {
	ID             string `json:"id"`
	ActorCode      string `json:"actor_code"`
	ActorType      string `json:"actor_type"`
	StaffAccountID string `json:"staff_account_id"`
	DisplayName    string `json:"display_name"`
	Status         string `json:"status"`
	Presence       string `json:"presence"`
	MaxActiveCases int    `json:"max_active_cases"`
}

type Queue struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	MembershipRole string `json:"membership_role"`
}

type MeResponse struct {
	Staff  staff.Account `json:"staff"`
	Actor  Actor         `json:"actor"`
	Queues []Queue       `json:"queues"`
}

type UpdatePresenceRequest struct {
	Presence string `json:"presence"`
}

type ReplyRequest struct {
	Message    string `json:"message"`
	Visibility string `json:"visibility"`
}

type EscalateRequest struct {
	QueueCode string `json:"queue_code"`
}

type EscalationResult struct {
	CaseID    string `json:"case_id"`
	Status    string `json:"status"`
	QueueCode string `json:"queue_code"`
}
