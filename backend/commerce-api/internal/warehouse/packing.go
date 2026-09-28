package warehouse

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

func (s *Service) MarkPacked(
	ctx context.Context,
	fulfillmentID string,
	request WarehouseActionRequest,
	actorID string,
) (Fulfillment, error) {
	return s.UpdateFulfillmentStatus(
		ctx,
		fulfillmentID,
		UpdateFulfillmentStatusRequest{
			Status: FulfillmentStatusPacked,

			Message: request.Message,
		},
		actorID,
	)
}

func (s *Service) MarkReadyForHandoff(
	ctx context.Context,
	fulfillmentID string,
	request WarehouseActionRequest,
	actorID string,
) (Fulfillment, error) {
	return s.UpdateFulfillmentStatus(
		ctx,
		fulfillmentID,
		UpdateFulfillmentStatusRequest{
			Status: FulfillmentStatusReadyForHandoff,

			Message: request.Message,
		},
		actorID,
	)
}

func (s *Service) CancelFulfillment(
	ctx context.Context,
	fulfillmentID string,
	request WarehouseActionRequest,
	actorID string,
) (Fulfillment, error) {
	fulfillmentID =
		strings.TrimSpace(
			fulfillmentID,
		)

	request.Message =
		strings.TrimSpace(
			request.Message,
		)

	actorID =
		normalizeActorID(
			actorID,
		)

	if !uuidPattern.MatchString(
		fulfillmentID,
	) ||
		utf8.RuneCountInString(
			request.Message,
		) > 1000 {
		return Fulfillment{},
			ErrInvalidInput
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Fulfillment{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err :=
		s.repository.LockFulfillmentTx(
			ctx,
			tx,
			fulfillmentID,
		)
	if err != nil {
		return Fulfillment{}, err
	}

	if current.Status ==
		FulfillmentStatusCancelled {
		return current, nil
	}

	if !allowedManualFulfillmentTransition(
		current.Status,
		FulfillmentStatusCancelled,
	) {
		return Fulfillment{},
			ErrFulfillmentTransitionNotAllowed
	}

	// A partial handoff leaves the fulfillment itself in
	// ready_for_handoff while shipment_fulfillments already contains
	// the physically assigned quantity.
	//
	// Never allow the manual cancel endpoint to hide that custody.
	assignedQuantity, err :=
		s.repository.AssignedQuantityTx(
			ctx,
			tx,
			current.ID,
		)
	if err != nil {
		return Fulfillment{}, err
	}

	if assignedQuantity > 0 {
		return Fulfillment{},
			ErrFulfillmentTransitionNotAllowed
	}

	now :=
		time.Now().UTC()

	if err :=
		s.repository.UpdateFulfillmentStatusTx(
			ctx,
			tx,
			current.ID,
			FulfillmentStatusCancelled,
			now,
		); err != nil {
		return Fulfillment{}, err
	}

	message :=
		request.Message

	if message == "" {
		message =
			"Warehouse fulfillment cancelled"
	}

	if err :=
		s.repository.InsertFulfillmentEventTx(
			ctx,
			tx,
			current.ID,
			"status_changed",
			current.Status,
			FulfillmentStatusCancelled,
			message,
			"admin",
			actorID,
		); err != nil {
		return Fulfillment{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Fulfillment{},
			fmt.Errorf(
				"commit warehouse fulfillment cancellation: %w",
				err,
			)
	}

	return s.repository.GetFulfillment(
		ctx,
		current.ID,
	)
}
