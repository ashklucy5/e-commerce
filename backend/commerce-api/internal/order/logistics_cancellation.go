package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type logisticsCancellationShipment struct {
	ID     string
	Status string
}

type logisticsCancellationFulfillment struct {
	ID     string
	Status string
}

type logisticsCancellationPlan struct {
	Shipment *logisticsCancellationShipment

	Fulfillments []logisticsCancellationFulfillment
}

func shipmentStatusAllowsOrderCancellation(
	status string,
) bool {
	switch status {
	case "pending",
		"cancelled":
		return true

	default:
		return false
	}
}

func warehouseFulfillmentStatusAllowsOrderCancellation(
	status string,
) bool {
	switch status {
	case "allocated",
		"waiting_inbound",
		"received",
		"picking",
		"packed",
		"ready_for_handoff":
		return true

	default:
		return false
	}
}

// PreparePreDispatchLogisticsCancellationTx locks and validates all
// Logistics state associated with an Order.
//
// Lock order:
//
//	order row                 -- already locked by caller
//	shipment row
//	warehouse fulfillment rows
//
// A warehouse handoff or shipment_fulfillment assignment means custody
// has already crossed the Warehouse boundary. Ordinary pre-dispatch
// cancellation must not undo that physical movement.
func (r *Repository) PreparePreDispatchLogisticsCancellationTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (logisticsCancellationPlan, error) {
	var plan logisticsCancellationPlan

	// ---------------------------------------------------------
	// Shipment
	//
	// Logistics v0 intentionally still has one shipment per Order.
	// This query will be refactored to a list when multi-shipment
	// support is enabled.
	// ---------------------------------------------------------

	var shipment logisticsCancellationShipment

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					status
				FROM shipments
				WHERE order_id = $1::uuid
				FOR UPDATE
			`,
			orderID,
		).Scan(
			&shipment.ID,
			&shipment.Status,
		)

	switch {
	case errors.Is(
		err,
		pgx.ErrNoRows,
	):
		// An Order may legitimately have no shipment yet.

	case err != nil:
		return logisticsCancellationPlan{},
			fmt.Errorf(
				"lock order shipment for cancellation: %w",
				err,
			)

	default:
		if !shipmentStatusAllowsOrderCancellation(
			shipment.Status,
		) {
			return logisticsCancellationPlan{},
				ErrCancellationNotAllowed
		}

		plan.Shipment =
			&shipment
	}

	// ---------------------------------------------------------
	// Warehouse fulfillments
	// ---------------------------------------------------------

	rows, err :=
		tx.Query(
			ctx,
			`
				SELECT
					id::text,
					status
				FROM warehouse_fulfillments
				WHERE
					order_id = $1::uuid
					AND status <> 'cancelled'
				ORDER BY id
				FOR UPDATE
			`,
			orderID,
		)
	if err != nil {
		return logisticsCancellationPlan{},
			fmt.Errorf(
				"lock warehouse fulfillments for order cancellation: %w",
				err,
			)
	}

	fulfillments :=
		make(
			[]logisticsCancellationFulfillment,
			0,
		)

	for rows.Next() {
		var item logisticsCancellationFulfillment

		if err :=
			rows.Scan(
				&item.ID,
				&item.Status,
			); err != nil {
			rows.Close()

			return logisticsCancellationPlan{},
				fmt.Errorf(
					"scan warehouse fulfillment for cancellation: %w",
					err,
				)
		}

		if !warehouseFulfillmentStatusAllowsOrderCancellation(
			item.Status,
		) {
			rows.Close()

			return logisticsCancellationPlan{},
				ErrCancellationNotAllowed
		}

		fulfillments =
			append(
				fulfillments,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {
		rows.Close()

		return logisticsCancellationPlan{},
			fmt.Errorf(
				"iterate warehouse fulfillments for cancellation: %w",
				err,
			)
	}

	rows.Close()

	plan.Fulfillments =
		fulfillments

	// ---------------------------------------------------------
	// Physical-custody barrier
	//
	// shipment_fulfillments is created by the Warehouse handoff
	// transaction. warehouse_handoffs is the corresponding history.
	//
	// Either one existing means ordinary cancellation must stop.
	// ---------------------------------------------------------

	var custodyTransferred bool

	err =
		tx.QueryRow(
			ctx,
			`
				SELECT
					EXISTS (
						SELECT 1

						FROM shipment_fulfillments sf

						JOIN warehouse_fulfillments wf
							ON wf.id = sf.fulfillment_id

						WHERE wf.order_id = $1::uuid
					)
					OR
					EXISTS (
						SELECT 1

						FROM warehouse_handoffs wh

						JOIN shipments s
							ON s.id = wh.shipment_id

						WHERE s.order_id = $1::uuid
					)
			`,
			orderID,
		).Scan(
			&custodyTransferred,
		)
	if err != nil {
		return logisticsCancellationPlan{},
			fmt.Errorf(
				"check logistics custody before cancellation: %w",
				err,
			)
	}

	if custodyTransferred {
		return logisticsCancellationPlan{},
			ErrCancellationNotAllowed
	}

	return plan, nil
}

// ApplyPreDispatchLogisticsCancellationTx applies the plan while all
// relevant rows remain locked by the caller's transaction.
func (r *Repository) ApplyPreDispatchLogisticsCancellationTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	customerID string,
	reason string,
	plan logisticsCancellationPlan,
	now time.Time,
) error {
	// ---------------------------------------------------------
	// Warehouse fulfillments
	// ---------------------------------------------------------

	for _, fulfillment := range plan.Fulfillments {

		tag, err :=
			tx.Exec(
				ctx,
				`
					UPDATE warehouse_fulfillments

					SET
						status = 'cancelled',
						updated_at = now()

					WHERE
						id = $1::uuid
						AND status = $2::varchar(40)
				`,
				fulfillment.ID,
				fulfillment.Status,
			)
		if err != nil {
			return fmt.Errorf(
				"cancel warehouse fulfillment with order: %w",
				err,
			)
		}

		if tag.RowsAffected() != 1 {
			return ErrCancellationNotAllowed
		}

		_, err =
			tx.Exec(
				ctx,
				`
					INSERT INTO warehouse_fulfillment_events (
						fulfillment_id,
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
						'order_cancelled',
						$2,
						'cancelled',
						$3,
						'order',
						$4,
						$5::timestamptz
					)
				`,
				fulfillment.ID,
				fulfillment.Status,
				reason,
				orderID,
				now,
			)
		if err != nil {
			return fmt.Errorf(
				"insert warehouse cancellation event: %w",
				err,
			)
		}
	}

	// ---------------------------------------------------------
	// Pending domestic shipment
	// ---------------------------------------------------------

	if plan.Shipment == nil {
		return nil
	}

	// An already-cancelled shipment is an idempotent no-op.
	if plan.Shipment.Status ==
		"cancelled" {

		_, err :=
			tx.Exec(
				ctx,
				`
					UPDATE delivery_otp_challenges

					SET
						status = 'cancelled',
						updated_at = now()

					WHERE
						shipment_id = $1::uuid
						AND status IN (
							'pending',
							'verified'
						)
				`,
				plan.Shipment.ID,
			)
		if err != nil {
			return fmt.Errorf(
				"cancel delivery OTP challenges: %w",
				err,
			)
		}

		return nil
	}

	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE shipments

				SET
					status = 'cancelled',
					updated_at = now()

				WHERE
					id = $1::uuid
					AND status = 'pending'
			`,
			plan.Shipment.ID,
		)
	if err != nil {
		return fmt.Errorf(
			"cancel pending shipment with order: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrCancellationNotAllowed
	}

	// Any unused verification challenge is no longer valid once
	// its shipment has been cancelled.
	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE delivery_otp_challenges

				SET
					status = 'cancelled',
					updated_at = now()

				WHERE
					shipment_id = $1::uuid
					AND status IN (
						'pending',
						'verified'
					)
			`,
			plan.Shipment.ID,
		)
	if err != nil {
		return fmt.Errorf(
			"cancel delivery OTP challenges: %w",
			err,
		)
	}

	// Keep Delivery's canonical tracking history complete.
	_, err =
		tx.Exec(
			ctx,
			`
				INSERT INTO delivery_tracking_events (
					shipment_id,
					order_id,
					source,
					event_code,
					status,
					message,
					occurred_at,
					created_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					'customer',
					'shipment_cancelled',
					'cancelled',
					$3,
					$4::timestamptz,
					now()
				)
			`,
			plan.Shipment.ID,
			orderID,
			reason,
			now,
		)
	if err != nil {
		return fmt.Errorf(
			"insert shipment cancellation tracking event: %w",
			err,
		)
	}

	_ = customerID

	return nil
}
