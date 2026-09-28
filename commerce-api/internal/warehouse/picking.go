package warehouse

import (
	"context"
	"fmt"
	"strings"
)

type WarehouseActionRequest struct {
	Message string `json:"message"`
}

func (s *Service) GetFulfillment(
	ctx context.Context,
	fulfillmentID string,
) (Fulfillment, error) {
	fulfillmentID = strings.TrimSpace(
		fulfillmentID,
	)

	if !uuidPattern.MatchString(
		fulfillmentID,
	) {
		return Fulfillment{},
			ErrFulfillmentNotFound
	}

	return s.repository.GetFulfillment(
		ctx,
		fulfillmentID,
	)
}

func (s *Service) ListFulfillmentEvents(
	ctx context.Context,
	fulfillmentID string,
) ([]FulfillmentEvent, error) {
	fulfillmentID = strings.TrimSpace(
		fulfillmentID,
	)

	if !uuidPattern.MatchString(
		fulfillmentID,
	) {
		return nil,
			ErrFulfillmentNotFound
	}

	if _, err :=
		s.repository.GetFulfillment(
			ctx,
			fulfillmentID,
		); err != nil {
		return nil, err
	}

	return s.repository.ListFulfillmentEvents(
		ctx,
		fulfillmentID,
	)
}

func (s *Service) StartPicking(
	ctx context.Context,
	fulfillmentID string,
	request WarehouseActionRequest,
	actorID string,
) (Fulfillment, error) {
	return s.UpdateFulfillmentStatus(
		ctx,
		fulfillmentID,
		UpdateFulfillmentStatusRequest{
			Status:  FulfillmentStatusPicking,
			Message: request.Message,
		},
		actorID,
	)
}

func (r *Repository) ListFulfillmentEvents(
	ctx context.Context,
	fulfillmentID string,
) ([]FulfillmentEvent, error) {
	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				id::text,
				fulfillment_id::text,
				event_type,
				COALESCE(from_status, ''),
				COALESCE(to_status, ''),
				COALESCE(message, ''),
				COALESCE(actor_type, ''),
				COALESCE(actor_id, ''),
				created_at
			FROM warehouse_fulfillment_events
			WHERE fulfillment_id = $1::uuid
			ORDER BY
				created_at ASC,
				id ASC
		`,
		fulfillmentID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list fulfillment events: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]FulfillmentEvent,
		0,
	)

	for rows.Next() {
		var event FulfillmentEvent

		if err := rows.Scan(
			&event.ID,
			&event.FulfillmentID,
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
					"scan fulfillment event: %w",
					err,
				)
		}

		items = append(
			items,
			event,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate fulfillment events: %w",
				err,
			)
	}

	return items, nil
}
