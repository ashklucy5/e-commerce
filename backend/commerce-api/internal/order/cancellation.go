package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/inventory"
)

type cancellationRefundLifecycle interface {
	RequestCancellationTx(
		ctx context.Context,
		tx pgx.Tx,
		orderID string,
		reason string,
		actorID string,
	) (string, error)
}

func (s *Service) SetCancellationRefundLifecycle(
	lifecycle cancellationRefundLifecycle,
) {
	s.cancellationRefunds = lifecycle
}

func (s *Service) CancelForCustomer(
	ctx context.Context,
	customerID string,
	orderID string,
	request CancelRequest,
) (Order, error) {
	if _, err :=
		s.GetForCustomer(
			ctx,
			customerID,
			orderID,
		); err != nil {
		return Order{}, err
	}

	return s.Cancel(
		ctx,
		orderID,
		request,
	)
}

func (s *Service) Cancel(
	ctx context.Context,
	orderID string,
	request CancelRequest,
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

	request.Reason =
		strings.TrimSpace(
			request.Reason,
		)

	if request.Reason == "" ||
		len(request.Reason) > 500 {
		return Order{},
			ErrCancellationReasonRequired
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

	// Cancellation is idempotent.
	if current.Status ==
		StatusCancelled {
		return current, nil
	}

	fromStatus :=
		current.Status

	// ---------------------------------------------------------
	// Validate cancellation before changing logistics,
	// inventory or payment state.
	// ---------------------------------------------------------

	switch current.Status {
	case StatusPendingPayment:
		if current.PaymentStatus !=
			PaymentStatusPending {
			return Order{},
				ErrCancellationNotAllowed
		}

	case StatusAwaitingProcurement:
		// awaiting_procurement is sourcing-only.
		//
		// No stock has been committed yet.
		if current.OrderType !=
			OrderTypeSourcing {
			return Order{},
				ErrCancellationNotAllowed
		}

		// Sourcing COD has not collected payment and has no
		// committed stock yet.
		if current.PaymentMethod ==
			PaymentMethodCOD &&
			current.PaymentStatus ==
				PaymentStatusCODPending {
			break
		}

		// Paid sourcing orders require the ordinary refund
		// lifecycle, but do not restore inventory because
		// procurement has not completed.
		if current.PaymentStatus ==
			PaymentStatusPaid {

			if current.PaymentMethod ==
				PaymentMethodCOD {
				return Order{},
					ErrCancellationNotAllowed
			}

			if s.cancellationRefunds == nil {
				return Order{},
					ErrRefundRequired
			}

			break
		}

		return Order{},
			ErrCancellationNotAllowed

	case StatusConfirmed,
		StatusProcessing:

		if current.PaymentMethod ==
			PaymentMethodCOD &&
			current.PaymentStatus ==
				PaymentStatusCODPending {
			break
		}

		if current.PaymentStatus ==
			PaymentStatusPaid {

			if current.PaymentMethod ==
				PaymentMethodCOD {
				return Order{},
					ErrCancellationNotAllowed
			}

			if s.cancellationRefunds == nil {
				return Order{},
					ErrRefundRequired
			}

			break
		}

		return Order{},
			ErrCancellationNotAllowed

	default:
		return Order{},
			ErrCancellationNotAllowed
	}

	// ---------------------------------------------------------
	// Lock and validate Warehouse + Delivery state.
	// ---------------------------------------------------------

	logisticsPlan, err :=
		s.repository.PreparePreDispatchLogisticsCancellationTx(
			ctx,
			tx,
			current.ID,
		)
	if err != nil {
		return Order{}, err
	}

	now :=
		time.Now().UTC()

	if err :=
		s.repository.ApplyPreDispatchLogisticsCancellationTx(
			ctx,
			tx,
			current.ID,
			current.CustomerID,
			request.Reason,
			logisticsPlan,
			now,
		); err != nil {
		return Order{}, err
	}

	// ---------------------------------------------------------
	// Inventory / payment cancellation.
	// ---------------------------------------------------------

	switch current.Status {
	case StatusPendingPayment:

		// Standard checkout orders have an inventory reservation.
		//
		// Sourcing orders are deliberately cartless and have no
		// inventory reservation before procurement.
		if current.OrderType !=
			OrderTypeSourcing {

			_, err :=
				s.inventory.ReleaseReferenceTx(
					ctx,
					tx,
					inventory.ReferenceActionInput{
						ReferenceType: inventoryReferenceType,

						ReferenceID: current.ID,

						Reason: "order_cancelled",

						Note: request.Reason,

						ActorType: "order",

						ActorID: current.ID,
					},
				)
			if err != nil {
				return Order{},
					fmt.Errorf(
						"release cancelled order inventory: %w",
						err,
					)
			}
		}

	case StatusAwaitingProcurement:

		// Stock has not been committed yet, so there is nothing
		// to release or restore.
		//
		// COD needs no refund. Paid online sourcing orders do.
		if current.PaymentMethod ==
			PaymentMethodCOD &&
			current.PaymentStatus ==
				PaymentStatusCODPending {
			break
		}

		if _, err :=
			s.cancellationRefunds.RequestCancellationTx(
				ctx,
				tx,
				current.ID,
				request.Reason,
				"customer",
			); err != nil {
			return Order{},
				fmt.Errorf(
					"create sourcing cancellation refund: %w",
					err,
				)
		}

	case StatusConfirmed,
		StatusProcessing:

		if current.PaymentMethod ==
			PaymentMethodCOD &&
			current.PaymentStatus ==
				PaymentStatusCODPending {

			restored, err :=
				s.inventory.RestoreCommittedReferenceTx(
					ctx,
					tx,
					inventory.ReferenceActionInput{
						ReferenceType: inventoryReferenceType,

						ReferenceID: current.ID,

						Reason: "order_cancelled_restock",

						Note: request.Reason,

						ActorType: "order",

						ActorID: current.ID,
					},
				)
			if err != nil {
				return Order{},
					fmt.Errorf(
						"restore cancelled COD inventory: %w",
						err,
					)
			}

			if restored !=
				current.ItemCount {
				return Order{},
					ErrInventoryReservationMismatch
			}

			break
		}

		restored, err :=
			s.inventory.RestoreCommittedReferenceTx(
				ctx,
				tx,
				inventory.ReferenceActionInput{
					ReferenceType: inventoryReferenceType,

					ReferenceID: current.ID,

					Reason: "paid_order_cancelled_restock",

					Note: request.Reason,

					ActorType: "order",

					ActorID: current.ID,
				},
			)
		if err != nil {
			return Order{},
				fmt.Errorf(
					"restore cancelled paid order inventory: %w",
					err,
				)
		}

		if restored !=
			current.ItemCount {
			return Order{},
				ErrInventoryReservationMismatch
		}

		if _, err :=
			s.cancellationRefunds.RequestCancellationTx(
				ctx,
				tx,
				current.ID,
				request.Reason,
				"customer",
			); err != nil {
			return Order{},
				fmt.Errorf(
					"create cancellation refund: %w",
					err,
				)
		}
	}

	if err :=
		s.repository.MarkCancelledTx(
			ctx,
			tx,
			current.ID,
			request.Reason,
			"customer",
			now,
		); err != nil {
		return Order{}, err
	}

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			current.ID,
			eventInsert{
				EventType: EventOrderCancelled,

				FromStatus: fromStatus,

				ToStatus: StatusCancelled,

				Message: request.Reason,

				ActorType: "customer",

				ActorID: current.CustomerID,
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
				"commit order cancellation: %w",
				err,
			)
	}

	return s.repository.GetByID(
		ctx,
		current.ID,
	)
}

func (r *Repository) MarkCancelledTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	reason string,
	cancelledBy string,
	cancelledAt time.Time,
) error {
	const query = `
		UPDATE orders
		SET
			status = 'cancelled',
			cancelled_at = $2,
			cancellation_reason = $3,
			cancelled_by = $4,
			updated_at = now()
		WHERE id = $1::uuid
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			orderID,
			cancelledAt,
			reason,
			cancelledBy,
		)
	if err != nil {
		return fmt.Errorf(
			"mark order cancelled: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOrderNotFound
	}

	return nil
}
