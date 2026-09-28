package order

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	EventOrderPlaced                  = "order_placed"
	EventPaymentConfirmed             = "payment_confirmed"
	EventPaymentExpired               = "payment_expired"
	EventOrderCancelled               = "order_cancelled"
	EventSourcingProcurementCompleted = "sourcing_procurement_completed"

	EventOrderProcessing = "order_processing"
	EventOrderShipped    = "order_shipped"
	EventOrderDelivered  = "order_delivered"
	EventOrderCompleted  = "order_completed"
)

type Event struct {
	ID string `json:"id"`

	OrderID string `json:"order_id"`

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

func (r *Repository) InsertEventTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	input eventInsert,
) error {
	const query = `
		INSERT INTO order_events (
			order_id,
			event_type,
			from_status,
			to_status,
			message,
			actor_type,
			actor_id,
			created_at
		)
		VALUES (
			$1::uuid,
			$2,
			NULLIF($3, ''),
			NULLIF($4, ''),
			NULLIF($5, ''),
			NULLIF($6, ''),
			NULLIF($7, ''),
			now()
		)
	`

	if _, err :=
		tx.Exec(
			ctx,
			query,
			orderID,
			input.EventType,
			input.FromStatus,
			input.ToStatus,
			input.Message,
			input.ActorType,
			input.ActorID,
		); err != nil {

		return fmt.Errorf(
			"insert order event: %w",
			err,
		)
	}

	/*
		The first order event is also the transactional boundary for
		the immutable Finance/Invoice commercial snapshot.

		At this point the order and order_items already exist inside
		the same PostgreSQL transaction.

		For a newly placed order we therefore freeze the authoritative
		product cost and create the persistent invoice before the
		transaction is allowed to commit.

		If any finance snapshot operation fails, this function returns
		an error and the surrounding order transaction rolls back.
	*/
	if input.EventType ==
		EventOrderPlaced {

		if err :=
			r.prepareNewOrderCommercialSnapshotTx(
				ctx,
				tx,
				orderID,
			); err != nil {

			return err
		}
	}

	/*
		Only selected customer-facing domain events produce
		notifications.

		The outbox rows are inserted using this same PostgreSQL
		transaction, so the order mutation/event, commercial snapshot,
		and notification cannot commit independently.
	*/
	if err :=
		r.enqueueOrderEventNotificationTx(
			ctx,
			tx,
			orderID,
			input,
		); err != nil {

		return err
	}

	return nil
}

func (r *Repository) ListEvents(
	ctx context.Context,
	orderID string,
) (
	[]Event,
	error,
) {
	const query = `
		SELECT
			id::text,
			order_id::text,
			event_type,
			COALESCE(from_status, ''),
			COALESCE(to_status, ''),
			COALESCE(message, ''),
			COALESCE(actor_type, ''),
			COALESCE(actor_id, ''),
			created_at

		FROM order_events

		WHERE
			order_id = $1::uuid

		ORDER BY
			created_at ASC,
			id ASC
	`

	rows, err :=
		r.db.Query(
			ctx,
			query,
			orderID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list order events: %w",
				err,
			)
	}

	defer rows.Close()

	result :=
		make(
			[]Event,
			0,
		)

	for rows.Next() {
		var event Event

		if err :=
			rows.Scan(
				&event.ID,
				&event.OrderID,
				&event.EventType,
				&event.FromStatus,
				&event.ToStatus,
				&event.Message,
				&event.ActorType,
				&event.ActorID,
				&event.CreatedAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan order event: %w",
					err,
				)
		}

		result =
			append(
				result,
				event,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate order events: %w",
				err,
			)
	}

	return result,
		nil
}
