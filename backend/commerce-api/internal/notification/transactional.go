package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type CustomerEventRequest struct {
	BaseDedupeKey string

	Category  string
	EventType string

	CustomerID string
	OrderID    string
	CaseID     string

	Phone string
	Email string

	TemplateKey string
	Message     string

	// Optional in-app presentation overrides. When omitted, the
	// notification package derives safe defaults from the event.
	Title     string
	ActionURL string

	Payload map[string]any
}

// EnqueueTx exposes the transactional outbox writer without requiring
// domain packages to construct or own a notification.Repository.
//
// The supplied PostgreSQL transaction remains owned by the caller.
func EnqueueTx(
	ctx context.Context,
	tx pgx.Tx,
	request EnqueueRequest,
) (
	OutboxItem,
	bool,
	error,
) {
	if tx == nil {
		return OutboxItem{},
			false,
			fmt.Errorf(
				"notification transaction is required",
			)
	}

	payload, err :=
		normalizeAndValidateEnqueueRequest(
			&request,
		)
	if err != nil {
		return OutboxItem{},
			false,
			err
	}

	// Repository.EnqueueTx only uses the supplied transaction.
	// It does not use Repository.db.
	repository :=
		&Repository{}

	return repository.EnqueueTx(
		ctx,
		tx,
		request,
		payload,
	)
}

// EnqueueCustomerEventTx creates independent SMS and email outbox rows.
//
// Missing recipients are intentionally still represented. The dispatcher
// records those rows as skipped/recipient_missing, giving us an auditable
// reason instead of silently losing the notification.
func EnqueueCustomerEventTx(
	ctx context.Context,
	tx pgx.Tx,
	request CustomerEventRequest,
) error {
	requests, err :=
		buildCustomerEventRequests(
			request,
		)
	if err != nil {
		return err
	}

	// The in-app inbox is durable customer state, not a delivery
	// channel. It is created in the same domain transaction as the
	// SMS/email outbox rows so all customer-facing notifications share
	// one event truth. Guest orders simply skip the inbox row.
	if request.CustomerID != "" {
		title, actionURL := customerInboxPresentation(
			request,
		)

		if err := EnqueueCustomerInboxTx(
			ctx,
			tx,
			CustomerInboxRequest{
				DedupeKey: request.BaseDedupeKey +
					":in_app",

				CustomerID: request.CustomerID,

				Category: request.Category,

				EventType: request.EventType,

				Title: title,

				Message: request.Message,

				ActionURL: actionURL,

				OrderID: request.OrderID,

				CaseID: request.CaseID,

				Metadata: request.Payload,
			},
		); err != nil {
			return err
		}
	}

	for _, item := range requests {
		if _, _, err :=
			EnqueueTx(
				ctx,
				tx,
				item,
			); err != nil {

			return err
		}
	}

	return nil
}

func buildCustomerEventRequests(
	request CustomerEventRequest,
) (
	[]EnqueueRequest,
	error,
) {
	request.BaseDedupeKey =
		strings.TrimSpace(
			request.BaseDedupeKey,
		)

	request.Category =
		strings.ToLower(
			strings.TrimSpace(
				request.Category,
			),
		)

	request.EventType =
		strings.ToLower(
			strings.TrimSpace(
				request.EventType,
			),
		)

	request.CustomerID =
		strings.TrimSpace(
			request.CustomerID,
		)

	request.OrderID =
		strings.TrimSpace(
			request.OrderID,
		)

	request.CaseID =
		strings.TrimSpace(
			request.CaseID,
		)

	request.Phone =
		strings.TrimSpace(
			request.Phone,
		)

	request.Email =
		strings.TrimSpace(
			request.Email,
		)

	request.TemplateKey =
		strings.ToLower(
			strings.TrimSpace(
				request.TemplateKey,
			),
		)

	request.Message =
		strings.TrimSpace(
			request.Message,
		)

	if request.BaseDedupeKey == "" ||
		request.Message == "" ||
		len(request.Message) > 1000 {

		return nil,
			ErrInvalidInput
	}

	payload :=
		make(
			map[string]any,
			len(request.Payload)+1,
		)

	for key, value := range request.Payload {
		payload[key] =
			value
	}

	// Callers cannot override the authoritative rendered message
	// through Payload.
	payload["message"] =
		request.Message

	return []EnqueueRequest{
		{
			DedupeKey: ChannelDedupeKey(
				request.BaseDedupeKey,
				ChannelSMS,
			),

			Category: request.Category,

			EventType: request.EventType,

			Channel: ChannelSMS,

			CustomerID: request.CustomerID,

			OrderID: request.OrderID,

			CaseID: request.CaseID,

			Recipient: request.Phone,

			TemplateKey: request.TemplateKey,

			Payload: payload,
		},
		{
			DedupeKey: ChannelDedupeKey(
				request.BaseDedupeKey,
				ChannelEmail,
			),

			Category: request.Category,

			EventType: request.EventType,

			Channel: ChannelEmail,

			CustomerID: request.CustomerID,

			OrderID: request.OrderID,

			CaseID: request.CaseID,

			Recipient: request.Email,

			TemplateKey: request.TemplateKey,

			Payload: payload,
		},
	}, nil
}
