package order

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"project.local/commerce-api/internal/inventory"
)

func (s *Service) TransitionFulfillment(
	ctx context.Context,
	orderID string,
	request FulfillmentRequest,
	actorID string,
) (Order, error) {
	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return Order{},
			ErrInvalidOrderID
	}

	normalizeFulfillmentRequest(
		&request,
	)

	fromStatus, ok :=
		expectedFulfillmentSource(
			request.Status,
		)
	if !ok {
		return Order{},
			ErrInvalidFulfillmentStatus
	}

	if err :=
		validateFulfillmentRequest(
			request,
		); err != nil {
		return Order{}, err
	}

	actorID =
		strings.TrimSpace(
			actorID,
		)

	if actorID == "" {
		actorID =
			"development-admin"
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Order{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err :=
		s.repository.LockOrderByIDTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return Order{}, err
	}

	if current.Status ==
		request.Status {
		return current, nil
	}

	if current.Status !=
		fromStatus {
		return Order{},
			ErrFulfillmentTransitionNotAllowed
	}

	// awaiting_procurement -> confirmed is reserved exclusively
	// for sourcing orders.
	if request.Status ==
		StatusConfirmed &&
		current.OrderType !=
			OrderTypeSourcing {
		return Order{},
			ErrFulfillmentTransitionNotAllowed
	}

	if err :=
		validateFulfillmentPayment(
			current,
			request.Status,
		); err != nil {
		return Order{}, err
	}

	now :=
		time.Now().UTC()

	// ---------------------------------------------------------
	// Sourcing procurement completion.
	//
	// Inventory must already have been physically received using
	// the existing inventory adjustment/receipt workflow.
	//
	// We now reserve the exact immutable order quantities and
	// immediately commit them so the sourcing order enters the
	// same committed-stock state as an ordinary confirmed order.
	// ---------------------------------------------------------

	if request.Status ==
		StatusConfirmed {

		if len(current.Items) == 0 {
			return Order{},
				ErrInventoryReservationMismatch
		}

		quantities :=
			make(
				map[string]int,
			)

		for _, item := range current.Items {

			quantities[item.VariantID] +=
				item.Quantity
		}

		reserveItems :=
			make(
				[]inventory.ReserveItem,
				0,
				len(quantities),
			)

		for variantID, quantity := range quantities {

			reserveItems =
				append(
					reserveItems,
					inventory.ReserveItem{
						VariantID: variantID,

						Quantity: quantity,
					},
				)
		}

		reservations, err :=
			s.inventory.ReserveReferenceTx(
				ctx,
				tx,
				inventory.ReserveReferenceInput{
					ReferenceType: inventoryReferenceType,

					ReferenceID: current.ID,

					Items: reserveItems,

					ActorType: "admin",

					ActorID: actorID,
				},
			)
		if err != nil {
			if errors.Is(
				err,
				inventory.ErrInsufficientStock,
			) {
				return Order{},
					ErrInsufficientStock
			}

			return Order{},
				fmt.Errorf(
					"reserve sourced order inventory: %w",
					err,
				)
		}

		if len(reservations) !=
			len(reserveItems) {
			return Order{},
				ErrInventoryReservationMismatch
		}

		committed, err :=
			s.inventory.CommitReferenceTx(
				ctx,
				tx,
				inventory.ReferenceActionInput{
					ReferenceType: inventoryReferenceType,

					ReferenceID: current.ID,

					Reason: "sourcing_procurement_completed",

					ActorType: "admin",

					ActorID: actorID,
				},
			)
		if err != nil {
			return Order{},
				fmt.Errorf(
					"commit sourced order inventory: %w",
					err,
				)
		}

		if committed !=
			len(reserveItems) {
			return Order{},
				ErrInventoryReservationMismatch
		}
	}

	// Shipping and delivery are intentionally not handled here anymore.
	//
	// awaiting_procurement -> confirmed is the sourcing procurement
	// completion operation.
	//
	// confirmed -> processing remains an Order operation.
	// delivered -> completed remains an Order operation.
	//
	// processing -> shipped is owned by internal/delivery only after
	// warehouse handoff has completed.
	//
	// shipped -> delivered is owned by receipt confirmation so a provider
	// "delivered" event cannot collect COD or finalize the order by itself.
	if err :=
		s.repository.TransitionFulfillmentTx(
			ctx,
			tx,
			orderID,
			fromStatus,
			request.Status,
			now,
		); err != nil {
		return Order{}, err
	}

	eventType, message :=
		fulfillmentEvent(
			request,
		)

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			orderID,
			eventInsert{
				EventType: eventType,

				FromStatus: fromStatus,

				ToStatus: request.Status,

				Message: message,

				ActorType: "admin",

				ActorID: actorID,
			},
		); err != nil {
		return Order{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Order{},
			fmt.Errorf(
				"commit fulfillment transition: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		orderID,
	)
}

func normalizeFulfillmentRequest(
	request *FulfillmentRequest,
) {
	request.Status =
		strings.ToLower(
			strings.TrimSpace(
				request.Status,
			),
		)

	request.CourierName =
		strings.TrimSpace(
			request.CourierName,
		)

	request.CourierReference =
		strings.TrimSpace(
			request.CourierReference,
		)

	request.TrackingNumber =
		strings.TrimSpace(
			request.TrackingNumber,
		)

	request.TrackingURL =
		strings.TrimSpace(
			request.TrackingURL,
		)
}

func expectedFulfillmentSource(
	target string,
) (string, bool) {
	switch target {
	case StatusConfirmed:
		return StatusAwaitingProcurement,
			true

	case StatusProcessing:
		return StatusConfirmed,
			true

	case StatusCompleted:
		return StatusDelivered,
			true

	case StatusShipped,
		StatusDelivered:
		return "",
			false

	default:
		return "",
			false
	}
}

func validateFulfillmentRequest(
	request FulfillmentRequest,
) error {
	if len(
		request.CourierName,
	) > 120 ||
		len(
			request.CourierReference,
		) > 160 ||
		len(
			request.TrackingNumber,
		) > 160 ||
		len(
			request.TrackingURL,
		) > 1000 {
		return ErrInvalidFulfillmentDetails
	}

	if request.TrackingURL != "" {
		parsed, err :=
			url.ParseRequestURI(
				request.TrackingURL,
			)

		if err != nil ||
			parsed.Host == "" ||
			(parsed.Scheme != "http" &&
				parsed.Scheme != "https") {
			return ErrInvalidFulfillmentDetails
		}
	}

	return nil
}

func validateFulfillmentPayment(
	order Order,
	targetStatus string,
) error {
	if order.PaymentMethod ==
		PaymentMethodCOD {

		switch targetStatus {
		case StatusConfirmed,
			StatusProcessing:

			if order.PaymentStatus !=
				PaymentStatusCODPending {
				return ErrPaymentNotReadyForFulfillment
			}

		case StatusCompleted:
			if order.PaymentStatus !=
				PaymentStatusCODCollected {
				return ErrPaymentNotReadyForFulfillment
			}
		}

		return nil
	}

	if order.PaymentStatus !=
		PaymentStatusPaid {
		return ErrPaymentNotReadyForFulfillment
	}

	return nil
}

func fulfillmentEvent(
	request FulfillmentRequest,
) (string, string) {
	switch request.Status {
	case StatusConfirmed:
		return EventSourcingProcurementCompleted,
			"Sourcing procurement completed; order confirmed"

	case StatusProcessing:
		return EventOrderProcessing,
			"Order moved to processing"

	case StatusCompleted:
		return EventOrderCompleted,
			"Order completed"

	default:
		return "",
			"Order fulfillment updated"
	}
}
