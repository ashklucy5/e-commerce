package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/inventory"
)

// MarkPaidTx confirms a verified online/manual payment inside a
// transaction owned by the caller.
//
// It does not commit or roll back tx.
func (s *Service) MarkPaidTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	paidAt time.Time,
) error {
	if tx == nil {
		return fmt.Errorf(
			"mark paid transaction is required",
		)
	}

	orderID = strings.TrimSpace(
		orderID,
	)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return ErrInvalidOrderID
	}

	now := time.Now().UTC()

	if paidAt.IsZero() ||
		paidAt.After(now) {
		paidAt = now
	} else {
		paidAt = paidAt.UTC()
	}

	current, err :=
		s.repository.LockOrderByIDTx(
			ctx,
			tx,
			orderID,
		)
	if err != nil {
		return err
	}

	// Idempotent verified payment callback.
	if current.PaymentStatus ==
		PaymentStatusPaid {
		return nil
	}

	if current.PaymentMethod !=
		PaymentMethodBKash &&
		current.PaymentMethod !=
			PaymentMethodNagad &&
		current.PaymentMethod !=
			PaymentMethodRocket &&
		current.PaymentMethod !=
			PaymentMethodBankTransfer {
		return ErrInvalidPaymentMethod
	}

	if current.Status !=
		StatusPendingPayment ||
		current.PaymentStatus !=
			PaymentStatusPending {
		return ErrOrderNotPendingPayment
	}

	// A provider callback/manual verification may arrive after
	// payment_due_at, but if the verified payment itself happened
	// before the deadline, we can still accept it.
	if current.PaymentDueAt == nil ||
		paidAt.After(
			*current.PaymentDueAt,
		) {
		return ErrPaymentWindowExpired
	}

	targetStatus :=
		StatusConfirmed

	if current.OrderType ==
		OrderTypeSourcing {

		targetStatus =
			StatusAwaitingProcurement
	} else {
		committed, err :=
			s.inventory.CommitReferenceTx(
				ctx,
				tx,
				inventory.ReferenceActionInput{
					ReferenceType: inventoryReferenceType,

					ReferenceID: orderID,

					Reason: "payment_confirmed",

					ActorType: "payment",

					ActorID: orderID,
				},
			)
		if err != nil {
			return fmt.Errorf(
				"commit paid order inventory: %w",
				err,
			)
		}

		if committed !=
			current.ItemCount {
			return ErrInventoryReservationMismatch
		}
	}

	if err :=
		s.repository.MarkPaidTx(
			ctx,
			tx,
			orderID,
			paidAt,
			targetStatus,
		); err != nil {
		return err
	}

	if err :=
		s.repository.InsertEventTx(
			ctx,
			tx,
			orderID,
			eventInsert{
				EventType: EventPaymentConfirmed,

				FromStatus: StatusPendingPayment,

				ToStatus: targetStatus,

				Message: "Payment confirmed",

				ActorType: "payment",
			},
		); err != nil {
		return err
	}

	return nil
}
