package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {
	return &Repository{
		db: db,
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

type inventoryBalance struct {
	QuantityOnHand   int
	QuantityReserved int
	ReorderLevel     int
}

type movementInsert struct {
	VariantID             string
	ReservationID         string
	MovementType          string
	QuantityOnHandDelta   int
	QuantityReservedDelta int
	QuantityOnHandAfter   int
	QuantityReservedAfter int
	ReferenceType         string
	ReferenceID           string
	Reason                string
	Note                  string
	ActorType             string
	ActorID               string
}

func scanStock(
	row rowScanner,
) (StockItem, error) {
	var item StockItem

	err := row.Scan(
		&item.VariantID,
		&item.SKU,
		&item.ProductID,
		&item.ProductCode,
		&item.ProductName,
		&item.QuantityOnHand,
		&item.QuantityReserved,
		&item.ReorderLevel,
		&item.UpdatedAt,
	)
	if err != nil {
		return StockItem{}, err
	}

	item.AvailableQuantity =
		item.QuantityOnHand -
			item.QuantityReserved

	item.LowStock =
		item.AvailableQuantity <=
			item.ReorderLevel

	return item, nil
}

func ensureInventoryRow(
	ctx context.Context,
	tx pgx.Tx,
	variantID string,
) error {
	const query = `
		INSERT INTO inventory (
			variant_id
		)
		SELECT id
		FROM product_variants
		WHERE id = $1::uuid
		ON CONFLICT (variant_id)
		DO NOTHING
	`

	tag, err := tx.Exec(
		ctx,
		query,
		variantID,
	)
	if err != nil {
		return fmt.Errorf(
			"ensure inventory row: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		const existsQuery = `
			SELECT 1
			FROM product_variants
			WHERE id = $1::uuid
		`

		var exists int

		err := tx.QueryRow(
			ctx,
			existsQuery,
			variantID,
		).Scan(
			&exists,
		)

		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrNotFound
		}

		if err != nil {
			return fmt.Errorf(
				"check inventory variant: %w",
				err,
			)
		}
	}

	return nil
}

func lockInventory(
	ctx context.Context,
	tx pgx.Tx,
	variantID string,
) (inventoryBalance, error) {
	const query = `
		SELECT
			quantity_on_hand,
			quantity_reserved,
			reorder_level
		FROM inventory
		WHERE variant_id = $1::uuid
		FOR UPDATE
	`

	var balance inventoryBalance

	err := tx.QueryRow(
		ctx,
		query,
		variantID,
	).Scan(
		&balance.QuantityOnHand,
		&balance.QuantityReserved,
		&balance.ReorderLevel,
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return inventoryBalance{},
			ErrNotFound
	}

	if err != nil {
		return inventoryBalance{},
			fmt.Errorf(
				"lock inventory: %w",
				err,
			)
	}

	return balance, nil
}

func updateReserved(
	ctx context.Context,
	tx pgx.Tx,
	variantID string,
	quantityReserved int,
) error {
	const query = `
		UPDATE inventory
		SET
			quantity_reserved = $2,
			updated_at = now()
		WHERE variant_id = $1::uuid
	`

	if _, err := tx.Exec(
		ctx,
		query,
		variantID,
		quantityReserved,
	); err != nil {
		return fmt.Errorf(
			"update reserved inventory: %w",
			err,
		)
	}

	return nil
}

func getReservationForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	referenceType string,
	referenceID string,
	variantID string,
) (Reservation, bool, error) {
	const query = `
		SELECT
			id::text,
			variant_id::text,
			reference_type,
			reference_id,
			quantity,
			status,
			expires_at,
			created_at,
			updated_at
		FROM inventory_reservations
		WHERE
			reference_type = $1
			AND reference_id = $2
			AND variant_id = $3::uuid
		FOR UPDATE
	`

	var reservation Reservation

	err := tx.QueryRow(
		ctx,
		query,
		referenceType,
		referenceID,
		variantID,
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
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Reservation{},
			false,
			nil
	}

	if err != nil {
		return Reservation{},
			false,
			fmt.Errorf(
				"load inventory reservation: %w",
				err,
			)
	}

	return reservation,
		true,
		nil
}

func getReservationByIDForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	id string,
) (Reservation, bool, error) {
	const query = `
		SELECT
			id::text,
			variant_id::text,
			reference_type,
			reference_id,
			quantity,
			status,
			expires_at,
			created_at,
			updated_at
		FROM inventory_reservations
		WHERE id = $1::uuid
		FOR UPDATE
	`

	var reservation Reservation

	err := tx.QueryRow(
		ctx,
		query,
		id,
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
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Reservation{},
			false,
			nil
	}

	if err != nil {
		return Reservation{},
			false,
			fmt.Errorf(
				"load reservation by id: %w",
				err,
			)
	}

	return reservation,
		true,
		nil
}

func insertMovement(
	ctx context.Context,
	tx pgx.Tx,
	movement movementInsert,
) error {
	const query = `
		INSERT INTO inventory_movements (
			variant_id,
			reservation_id,
			movement_type,
			quantity_on_hand_delta,
			quantity_reserved_delta,
			quantity_on_hand_after,
			quantity_reserved_after,
			reference_type,
			reference_id,
			reason,
			note,
			actor_type,
			actor_id
		)
		VALUES (
			$1::uuid,
			NULLIF($2::text, '')::uuid,
			$3,
			$4,
			$5,
			$6,
			$7,
			NULLIF($8::text, ''),
			NULLIF($9::text, ''),
			NULLIF($10::text, ''),
			NULLIF($11::text, ''),
			NULLIF($12::text, ''),
			NULLIF($13::text, '')
		)
		RETURNING id::text
	`

	var movementID string

	if err := tx.QueryRow(
		ctx,
		query,
		movement.VariantID,
		movement.ReservationID,
		movement.MovementType,
		movement.QuantityOnHandDelta,
		movement.QuantityReservedDelta,
		movement.QuantityOnHandAfter,
		movement.QuantityReservedAfter,
		movement.ReferenceType,
		movement.ReferenceID,
		movement.Reason,
		movement.Note,
		movement.ActorType,
		movement.ActorID,
	).Scan(
		&movementID,
	); err != nil {

		return fmt.Errorf(
			"insert inventory movement: %w",
			err,
		)
	}

	if err :=
		enqueueLowStockCrossingNotificationTx(
			ctx,
			tx,
			movement,
			movementID,
		); err != nil {

		return err
	}

	return nil
}