package inventory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---------------------------------------------------------
// Service
// ---------------------------------------------------------

func (s *Service) ReserveReference(
	ctx context.Context,
	input ReserveReferenceInput,
) ([]Reservation, error) {
	if err := normalizeReserveReferenceInput(
		&input,
	); err != nil {
		return nil, err
	}

	return s.repository.ReserveReference(
		ctx,
		input,
	)
}

func (s *Service) ReserveReferenceTx(
	ctx context.Context,
	tx pgx.Tx,
	input ReserveReferenceInput,
) ([]Reservation, error) {
	if tx == nil {
		return nil, ErrInvalidInput
	}

	if err := normalizeReserveReferenceInput(
		&input,
	); err != nil {
		return nil, err
	}

	return s.repository.reserveReferenceTx(
		ctx,
		tx,
		input,
	)
}

func (s *Service) ReleaseReference(
	ctx context.Context,
	input ReferenceActionInput,
) (int, error) {
	if err := normalizeReferenceAction(
		&input,
	); err != nil {
		return 0, err
	}

	return s.repository.ReleaseReference(
		ctx,
		input,
	)
}

func (s *Service) ReleaseReferenceTx(
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

	return s.repository.closeReferenceTx(
		ctx,
		tx,
		input,
		"released",
		"release",
		false,
	)
}

func (s *Service) CommitReference(
	ctx context.Context,
	input ReferenceActionInput,
) (int, error) {
	if err := normalizeReferenceAction(
		&input,
	); err != nil {
		return 0, err
	}

	return s.repository.CommitReference(
		ctx,
		input,
	)
}

func (s *Service) CommitReferenceTx(
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

	return s.repository.closeReferenceTx(
		ctx,
		tx,
		input,
		"committed",
		"commit",
		true,
	)
}

func (s *Service) ExpireDue(
	ctx context.Context,
	now time.Time,
	limit int,
) (int, error) {
	if limit <= 0 {
		limit = 100
	}

	if limit > 1000 {
		limit = 1000
	}

	return s.repository.ExpireDue(
		ctx,
		now,
		limit,
	)
}

// ---------------------------------------------------------
// Normalization
// ---------------------------------------------------------

func normalizeReserveReferenceInput(
	input *ReserveReferenceInput,
) error {
	input.ReferenceType =
		strings.TrimSpace(
			input.ReferenceType,
		)

	input.ReferenceID =
		strings.TrimSpace(
			input.ReferenceID,
		)

	input.ActorType =
		strings.TrimSpace(
			input.ActorType,
		)

	input.ActorID =
		strings.TrimSpace(
			input.ActorID,
		)

	if input.ReferenceType == "" ||
		input.ReferenceID == "" ||
		len(input.Items) == 0 ||
		!validActorPair(
			input.ActorType,
			input.ActorID,
		) {
		return ErrInvalidInput
	}

	if input.ExpiresAt != nil &&
		!input.ExpiresAt.After(
			time.Now(),
		) {
		return ErrInvalidInput
	}

	quantities := make(
		map[string]int,
	)

	for _, item := range input.Items {
		variantID :=
			strings.TrimSpace(
				item.VariantID,
			)

		if variantID == "" ||
			item.Quantity <= 0 {
			return ErrInvalidInput
		}

		quantities[variantID] +=
			item.Quantity
	}

	variantIDs := make(
		[]string,
		0,
		len(quantities),
	)

	for variantID := range quantities {
		variantIDs = append(
			variantIDs,
			variantID,
		)
	}

	sort.Strings(
		variantIDs,
	)

	input.Items = make(
		[]ReserveItem,
		0,
		len(variantIDs),
	)

	for _, variantID := range variantIDs {
		input.Items = append(
			input.Items,
			ReserveItem{
				VariantID: variantID,
				Quantity:  quantities[variantID],
			},
		)
	}

	return nil
}

func normalizeReferenceAction(
	input *ReferenceActionInput,
) error {
	input.ReferenceType =
		strings.TrimSpace(
			input.ReferenceType,
		)

	input.ReferenceID =
		strings.TrimSpace(
			input.ReferenceID,
		)

	input.Reason =
		strings.TrimSpace(
			input.Reason,
		)

	input.Note =
		strings.TrimSpace(
			input.Note,
		)

	input.ActorType =
		strings.TrimSpace(
			input.ActorType,
		)

	input.ActorID =
		strings.TrimSpace(
			input.ActorID,
		)

	if input.ReferenceType == "" ||
		input.ReferenceID == "" ||
		!validActorPair(
			input.ActorType,
			input.ActorID,
		) {
		return ErrInvalidInput
	}

	return nil
}

// ---------------------------------------------------------
// Repository - reserve
// ---------------------------------------------------------

func (r *Repository) ReserveReference(
	ctx context.Context,
	input ReserveReferenceInput,
) ([]Reservation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"begin inventory reservation: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	result, err :=
		r.reserveReferenceTx(
			ctx,
			tx,
			input,
		)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf(
				"commit inventory reservation: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) reserveReferenceTx(
	ctx context.Context,
	tx pgx.Tx,
	input ReserveReferenceInput,
) ([]Reservation, error) {
	result := make(
		[]Reservation,
		0,
		len(input.Items),
	)

	for _, item := range input.Items {
		if err := ensureInventoryRow(
			ctx,
			tx,
			item.VariantID,
		); err != nil {
			return nil, err
		}

		balance, err :=
			lockInventory(
				ctx,
				tx,
				item.VariantID,
			)
		if err != nil {
			return nil, err
		}

		existing, exists, err :=
			getReservationForUpdate(
				ctx,
				tx,
				input.ReferenceType,
				input.ReferenceID,
				item.VariantID,
			)
		if err != nil {
			return nil, err
		}

		if exists {
			if existing.Status != "active" {
				return nil,
					ErrReservationClosed
			}

			delta :=
				item.Quantity -
					existing.Quantity

			if delta > 0 {
				available :=
					balance.QuantityOnHand -
						balance.QuantityReserved

				if delta > available {
					return nil,
						ErrInsufficientStock
				}
			}

			newReserved :=
				balance.QuantityReserved +
					delta

			if newReserved < 0 ||
				newReserved >
					balance.QuantityOnHand {
				return nil,
					ErrInventoryInvariant
			}

			if delta != 0 {
				if err := updateReserved(
					ctx,
					tx,
					item.VariantID,
					newReserved,
				); err != nil {
					return nil, err
				}
			}

			const updateReservationQuery = `
				UPDATE inventory_reservations
				SET
					quantity = $2,
					expires_at = $3,
					updated_at = now()
				WHERE id = $1::uuid
				RETURNING
					id::text,
					variant_id::text,
					reference_type,
					reference_id,
					quantity,
					status,
					expires_at,
					created_at,
					updated_at
			`

			var updated Reservation

			if err := tx.QueryRow(
				ctx,
				updateReservationQuery,
				existing.ID,
				item.Quantity,
				input.ExpiresAt,
			).Scan(
				&updated.ID,
				&updated.VariantID,
				&updated.ReferenceType,
				&updated.ReferenceID,
				&updated.Quantity,
				&updated.Status,
				&updated.ExpiresAt,
				&updated.CreatedAt,
				&updated.UpdatedAt,
			); err != nil {
				return nil,
					fmt.Errorf(
						"update inventory reservation: %w",
						err,
					)
			}

			if delta != 0 {
				if err := insertMovement(
					ctx,
					tx,
					movementInsert{
						VariantID: item.VariantID,

						ReservationID: updated.ID,

						MovementType: "reservation_change",

						QuantityOnHandDelta: 0,

						QuantityReservedDelta: delta,

						QuantityOnHandAfter: balance.QuantityOnHand,

						QuantityReservedAfter: newReserved,

						ReferenceType: input.ReferenceType,

						ReferenceID: input.ReferenceID,

						ActorType: input.ActorType,

						ActorID: input.ActorID,
					},
				); err != nil {
					return nil, err
				}
			}

			result = append(
				result,
				updated,
			)

			continue
		}

		available :=
			balance.QuantityOnHand -
				balance.QuantityReserved

		if item.Quantity > available {
			return nil,
				ErrInsufficientStock
		}

		const insertReservationQuery = `
			INSERT INTO inventory_reservations (
				variant_id,
				reference_type,
				reference_id,
				quantity,
				status,
				expires_at
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				$4,
				'active',
				$5
			)
			RETURNING
				id::text,
				variant_id::text,
				reference_type,
				reference_id,
				quantity,
				status,
				expires_at,
				created_at,
				updated_at
		`

		var reservation Reservation

		if err := tx.QueryRow(
			ctx,
			insertReservationQuery,
			item.VariantID,
			input.ReferenceType,
			input.ReferenceID,
			item.Quantity,
			input.ExpiresAt,
		).Scan(
			&reservation.ID,
			&reservation.VariantID,
			&reservation.ReferenceType,
			&reservation.ReferenceID,
			&reservation.Quantity,
			&reservation.Status,
			&reservation.ExpiresAt,
			&reservation.CreatedAt,
			&reservation.UpdatedAt,
		); err != nil {
			return nil,
				fmt.Errorf(
					"create inventory reservation: %w",
					err,
				)
		}

		newReserved :=
			balance.QuantityReserved +
				item.Quantity

		if err := updateReserved(
			ctx,
			tx,
			item.VariantID,
			newReserved,
		); err != nil {
			return nil, err
		}

		if err := insertMovement(
			ctx,
			tx,
			movementInsert{
				VariantID: item.VariantID,

				ReservationID: reservation.ID,

				MovementType: "reserve",

				QuantityOnHandDelta: 0,

				QuantityReservedDelta: item.Quantity,

				QuantityOnHandAfter: balance.QuantityOnHand,

				QuantityReservedAfter: newReserved,

				ReferenceType: input.ReferenceType,

				ReferenceID: input.ReferenceID,

				ActorType: input.ActorType,

				ActorID: input.ActorID,
			},
		); err != nil {
			return nil, err
		}

		result = append(
			result,
			reservation,
		)
	}

	return result, nil
}

// ---------------------------------------------------------
// Repository - release / commit
// ---------------------------------------------------------

func (r *Repository) ReleaseReference(
	ctx context.Context,
	input ReferenceActionInput,
) (int, error) {
	return r.closeReference(
		ctx,
		input,
		"released",
		"release",
		false,
	)
}

func (r *Repository) CommitReference(
	ctx context.Context,
	input ReferenceActionInput,
) (int, error) {
	return r.closeReference(
		ctx,
		input,
		"committed",
		"commit",
		true,
	)
}

func (r *Repository) closeReference(
	ctx context.Context,
	input ReferenceActionInput,
	status string,
	movementType string,
	removeOnHand bool,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0,
			fmt.Errorf(
				"begin inventory reference close: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	closed, err :=
		r.closeReferenceTx(
			ctx,
			tx,
			input,
			status,
			movementType,
			removeOnHand,
		)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0,
			fmt.Errorf(
				"commit inventory reference close: %w",
				err,
			)
	}

	return closed, nil
}

func (r *Repository) closeReferenceTx(
	ctx context.Context,
	tx pgx.Tx,
	input ReferenceActionInput,
	status string,
	movementType string,
	removeOnHand bool,
) (int, error) {
	const variantsQuery = `
		SELECT variant_id::text
		FROM inventory_reservations
		WHERE
			reference_type = $1
			AND reference_id = $2
			AND status = 'active'
	`

	rows, err := tx.Query(
		ctx,
		variantsQuery,
		input.ReferenceType,
		input.ReferenceID,
	)
	if err != nil {
		return 0,
			fmt.Errorf(
				"load active reservations: %w",
				err,
			)
	}

	var variantIDs []string

	for rows.Next() {
		var variantID string

		if err := rows.Scan(
			&variantID,
		); err != nil {
			rows.Close()

			return 0,
				fmt.Errorf(
					"scan reservation variant: %w",
					err,
				)
		}

		variantIDs = append(
			variantIDs,
			variantID,
		)
	}

	if err := rows.Err(); err != nil {
		rows.Close()

		return 0,
			fmt.Errorf(
				"iterate reservation variants: %w",
				err,
			)
	}

	rows.Close()

	sort.Strings(
		variantIDs,
	)

	closed := 0

	for _, variantID := range variantIDs {
		if err := ensureInventoryRow(
			ctx,
			tx,
			variantID,
		); err != nil {
			return 0, err
		}

		balance, err :=
			lockInventory(
				ctx,
				tx,
				variantID,
			)
		if err != nil {
			return 0, err
		}

		reservation, exists, err :=
			getReservationForUpdate(
				ctx,
				tx,
				input.ReferenceType,
				input.ReferenceID,
				variantID,
			)
		if err != nil {
			return 0, err
		}

		if !exists ||
			reservation.Status != "active" {
			continue
		}

		newReserved :=
			balance.QuantityReserved -
				reservation.Quantity

		if newReserved < 0 {
			return 0,
				ErrInventoryInvariant
		}

		newOnHand :=
			balance.QuantityOnHand

		onHandDelta := 0

		if removeOnHand {
			newOnHand -=
				reservation.Quantity

			onHandDelta =
				-reservation.Quantity

			if newOnHand < 0 ||
				newReserved > newOnHand {
				return 0,
					ErrInventoryInvariant
			}
		}

		const updateInventoryQuery = `
			UPDATE inventory
			SET
				quantity_on_hand = $2,
				quantity_reserved = $3,
				updated_at = now()
			WHERE variant_id = $1::uuid
		`

		if _, err := tx.Exec(
			ctx,
			updateInventoryQuery,
			variantID,
			newOnHand,
			newReserved,
		); err != nil {
			return 0,
				fmt.Errorf(
					"close reservation inventory update: %w",
					err,
				)
		}

		const updateReservationQuery = `
			UPDATE inventory_reservations
			SET
				status = $2,
				updated_at = now()
			WHERE id = $1::uuid
		`

		if _, err := tx.Exec(
			ctx,
			updateReservationQuery,
			reservation.ID,
			status,
		); err != nil {
			return 0,
				fmt.Errorf(
					"close reservation: %w",
					err,
				)
		}

		reason :=
			input.Reason

		if reason == "" {
			reason =
				movementType
		}

		if err := insertMovement(
			ctx,
			tx,
			movementInsert{
				VariantID: variantID,

				ReservationID: reservation.ID,

				MovementType: movementType,

				QuantityOnHandDelta: onHandDelta,

				QuantityReservedDelta: -reservation.Quantity,

				QuantityOnHandAfter: newOnHand,

				QuantityReservedAfter: newReserved,

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

		closed++
	}

	return closed, nil
}

// ---------------------------------------------------------
// Repository - expire
// ---------------------------------------------------------

func (r *Repository) ExpireDue(
	ctx context.Context,
	now time.Time,
	limit int,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0,
			fmt.Errorf(
				"begin inventory expiry: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const query = `
		SELECT
			id::text,
			variant_id::text
		FROM inventory_reservations
		WHERE
			status = 'active'
			AND expires_at IS NOT NULL
			AND expires_at <= $1
		ORDER BY
			variant_id,
			id
		LIMIT $2
	`

	rows, err := tx.Query(
		ctx,
		query,
		now,
		limit,
	)
	if err != nil {
		return 0,
			fmt.Errorf(
				"load expired reservations: %w",
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
					"scan expired reservation: %w",
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
				"iterate expired reservations: %w",
				err,
			)
	}

	rows.Close()

	expired := 0

	for _, candidate := range candidates {
		if err := ensureInventoryRow(
			ctx,
			tx,
			candidate.VariantID,
		); err != nil {
			return 0, err
		}

		balance, err :=
			lockInventory(
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
			reservation.Status != "active" ||
			reservation.ExpiresAt == nil ||
			reservation.ExpiresAt.After(now) {
			continue
		}

		newReserved :=
			balance.QuantityReserved -
				reservation.Quantity

		if newReserved < 0 {
			return 0,
				ErrInventoryInvariant
		}

		if err := updateReserved(
			ctx,
			tx,
			candidate.VariantID,
			newReserved,
		); err != nil {
			return 0, err
		}

		const updateReservationQuery = `
			UPDATE inventory_reservations
			SET
				status = 'expired',
				updated_at = now()
			WHERE id = $1::uuid
		`

		if _, err := tx.Exec(
			ctx,
			updateReservationQuery,
			reservation.ID,
		); err != nil {
			return 0,
				fmt.Errorf(
					"expire reservation: %w",
					err,
				)
		}

		if err := insertMovement(
			ctx,
			tx,
			movementInsert{
				VariantID: reservation.VariantID,

				ReservationID: reservation.ID,

				MovementType: "expire",

				QuantityOnHandDelta: 0,

				QuantityReservedDelta: -reservation.Quantity,

				QuantityOnHandAfter: balance.QuantityOnHand,

				QuantityReservedAfter: newReserved,

				ReferenceType: reservation.ReferenceType,

				ReferenceID: reservation.ReferenceID,

				Reason: "reservation_expired",
			},
		); err != nil {
			return 0, err
		}

		expired++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0,
			fmt.Errorf(
				"commit inventory expiry: %w",
				err,
			)
	}

	return expired, nil
}
