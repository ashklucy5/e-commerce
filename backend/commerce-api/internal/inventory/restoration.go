package inventory

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

const maxPostgresInteger = 2147483647

func (s *Service) RestoreCommittedReferenceTx(
	ctx context.Context,
	tx pgx.Tx,
	input ReferenceActionInput,
) (int, error) {
	if tx == nil {
		return 0, ErrInvalidInput
	}

	if err := normalizeReferenceAction(
		&input,
	); err != nil {
		return 0, err
	}

	return s.repository.restoreCommittedReferenceTx(
		ctx,
		tx,
		input,
	)
}

func (r *Repository) restoreCommittedReferenceTx(
	ctx context.Context,
	tx pgx.Tx,
	input ReferenceActionInput,
) (int, error) {
	const query = `
		SELECT
			id::text,
			variant_id::text
		FROM inventory_reservations
		WHERE
			reference_type = $1
			AND reference_id = $2
			AND status = 'committed'
		ORDER BY
			variant_id,
			id
	`

	rows, err := tx.Query(
		ctx,
		query,
		input.ReferenceType,
		input.ReferenceID,
	)
	if err != nil {
		return 0,
			fmt.Errorf(
				"load committed reservations: %w",
				err,
			)
	}

	type candidate struct {
		ID        string
		VariantID string
	}

	candidates := make(
		[]candidate,
		0,
	)

	for rows.Next() {
		var item candidate

		if err := rows.Scan(
			&item.ID,
			&item.VariantID,
		); err != nil {
			rows.Close()

			return 0,
				fmt.Errorf(
					"scan committed reservation: %w",
					err,
				)
		}

		candidates = append(
			candidates,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		rows.Close()

		return 0,
			fmt.Errorf(
				"iterate committed reservations: %w",
				err,
			)
	}

	rows.Close()

	sort.Slice(
		candidates,
		func(i int, j int) bool {
			if candidates[i].VariantID ==
				candidates[j].VariantID {
				return candidates[i].ID <
					candidates[j].ID
			}

			return candidates[i].VariantID <
				candidates[j].VariantID
		},
	)

	restored := 0

	for _, candidate := range candidates {
		if err := ensureInventoryRow(
			ctx,
			tx,
			candidate.VariantID,
		); err != nil {
			return 0, err
		}

		balance, err := lockInventory(
			ctx,
			tx,
			candidate.VariantID,
		)
		if err != nil {
			return 0, err
		}

		reservation, exists, err :=
			getReservationByIDForUpdate(
				ctx,
				tx,
				candidate.ID,
			)
		if err != nil {
			return 0, err
		}

		if !exists ||
			reservation.Status != "committed" ||
			reservation.ReferenceType != input.ReferenceType ||
			reservation.ReferenceID != input.ReferenceID {
			continue
		}

		if reservation.Quantity >
			maxPostgresInteger-balance.QuantityOnHand {
			return 0,
				ErrInventoryInvariant
		}

		newOnHand :=
			balance.QuantityOnHand +
				reservation.Quantity

		const updateQuery = `
			UPDATE inventory
			SET
				quantity_on_hand = $2,
				updated_at = now()
			WHERE variant_id = $1::uuid
		`

		if _, err := tx.Exec(
			ctx,
			updateQuery,
			candidate.VariantID,
			newOnHand,
		); err != nil {
			return 0,
				fmt.Errorf(
					"restore committed inventory: %w",
					err,
				)
		}

		reason := input.Reason

		if reason == "" {
			reason =
				"restore_committed_inventory"
		}

		if err := insertMovement(
			ctx,
			tx,
			movementInsert{
				VariantID: candidate.VariantID,

				ReservationID: reservation.ID,

				MovementType: "adjustment",

				QuantityOnHandDelta: reservation.Quantity,

				QuantityReservedDelta: 0,

				QuantityOnHandAfter: newOnHand,

				QuantityReservedAfter: balance.QuantityReserved,

				ReferenceType: input.ReferenceType,

				ReferenceID: input.ReferenceID,

				Reason: reason,

				Note: input.Note,

				ActorType: input.ActorType,

				ActorID: input.ActorID,
			},
		); err != nil {
			return 0, err
		}

		restored++
	}

	return restored, nil
}
