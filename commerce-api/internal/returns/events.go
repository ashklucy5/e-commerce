package returns

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	EventRequested = "return_requested"
	EventApproved  = "return_approved"
	EventRejected  = "return_rejected"
	EventReceived  = "return_received"
	EventInspected = "return_inspected"
	EventCompleted = "return_completed"
	EventCancelled = "return_cancelled"
)

type Event struct {
	ID       string `json:"id"`
	ReturnID string `json:"return_id"`

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
	returnID string,
	input eventInsert,
) error {
	const query = `
		INSERT INTO return_events (
			return_id,
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

	if _, err := tx.Exec(
		ctx,
		query,
		returnID,
		input.EventType,
		input.FromStatus,
		input.ToStatus,
		input.Message,
		input.ActorType,
		input.ActorID,
	); err != nil {
		return fmt.Errorf(
			"insert return event: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ListEvents(
	ctx context.Context,
	returnID string,
) ([]Event, error) {
	const query = `
		SELECT
			id::text,
			return_id::text,
			event_type,
			COALESCE(from_status, ''),
			COALESCE(to_status, ''),
			COALESCE(message, ''),
			COALESCE(actor_type, ''),
			COALESCE(actor_id, ''),
			created_at
		FROM return_events
		WHERE return_id = $1::uuid
		ORDER BY
			created_at,
			id
	`

	rows, err := r.db.Query(
		ctx,
		query,
		returnID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list return events: %w",
				err,
			)
	}
	defer rows.Close()

	result := make(
		[]Event,
		0,
	)

	for rows.Next() {
		var event Event

		if err := rows.Scan(
			&event.ID,
			&event.ReturnID,
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
					"scan return event: %w",
					err,
				)
		}

		result = append(
			result,
			event,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate return events: %w",
				err,
			)
	}

	return result, nil
}
