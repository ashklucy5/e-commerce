package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/notification"
)

func enqueueLowStockCrossingNotificationTx(
	ctx context.Context,
	tx pgx.Tx,
	movement movementInsert,
	movementID string,
) error {
	beforeOnHand :=
		movement.QuantityOnHandAfter -
			movement.QuantityOnHandDelta

	beforeReserved :=
		movement.QuantityReservedAfter -
			movement.QuantityReservedDelta

	beforeAvailable :=
		beforeOnHand -
			beforeReserved

	afterAvailable :=
		movement.QuantityOnHandAfter -
			movement.QuantityReservedAfter

	var (
		sku          string
		productName  string
		reorderLevel int
	)

	if err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					v.sku,
					p.name,
					COALESCE(i.reorder_level, 0)
				FROM product_variants v
				JOIN products p
					ON p.id = v.product_id
				JOIN inventory i
					ON i.variant_id = v.id
				WHERE v.id = $1::uuid
			`,
			movement.VariantID,
		).Scan(
			&sku,
			&productName,
			&reorderLevel,
		); err != nil {

		return fmt.Errorf(
			"load low-stock notification context: %w",
			err,
		)
	}

	/*
		Notify only on a real threshold crossing:

		    before available > reorder level
		    after available <= reorder level

		Remaining below the threshold does not generate more
		notifications.

		If inventory later recovers above the threshold and then
		drops again, a new movement ID creates a new alert.
	*/
	if beforeAvailable <= reorderLevel ||
		afterAvailable > reorderLevel {

		return nil
	}

	priority :=
		notification.StaffPriorityAttention

	if afterAvailable <= 0 {
		priority =
			notification.StaffPriorityCritical
	}

	if err :=
		notification.EnqueueStaffEventTx(
			ctx,
			tx,
			notification.StaffEventRequest{
				DedupeKey: "staff:inventory:" +
					movement.VariantID +
					":low-stock:" +
					movementID,

				Category: notification.StaffCategoryInventory,

				EventType: notification.StaffEventInventoryLowStock,

				Priority: priority,

				Title: "Low stock",

				Message: fmt.Sprintf(
					"%s (%s) has %d available; reorder level is %d.",
					productName,
					sku,
					afterAvailable,
					reorderLevel,
				),

				ActionURL: "/admin/inventory",

				EntityType: "variant",

				EntityID: movement.VariantID,

				RequiredPermission: "admin.inventory.read",

				Metadata: map[string]any{
					"sku": sku,

					"product_name": productName,

					"available_quantity": afterAvailable,

					"reorder_level": reorderLevel,

					"movement_id": movementID,

					"movement_type": movement.MovementType,

					"reference_type": movement.ReferenceType,

					"reference_id": movement.ReferenceID,
				},
			},
		); err != nil {

		return fmt.Errorf(
			"enqueue low-stock staff notification: %w",
			err,
		)
	}

	return nil
}