package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Service) List(
	ctx context.Context,
	limit int,
	offset int,
) (StockListResult, error) {
	if limit <= 0 {
		limit = 50
	}

	if limit > 200 {
		limit = 200
	}

	if offset < 0 {
		offset = 0
	}

	return s.repository.List(
		ctx,
		limit,
		offset,
	)
}

func (s *Service) Get(
	ctx context.Context,
	variantID string,
) (StockItem, error) {
	variantID = strings.TrimSpace(
		variantID,
	)

	if variantID == "" {
		return StockItem{},
			ErrInvalidInput
	}

	return s.repository.Get(
		ctx,
		variantID,
	)
}

func (s *Service) Update(
	ctx context.Context,
	variantID string,
	request UpdateRequest,
) (StockItem, error) {
	variantID = strings.TrimSpace(
		variantID,
	)

	if variantID == "" ||
		request.ReorderLevel == nil ||
		*request.ReorderLevel < 0 {
		return StockItem{},
			ErrInvalidInput
	}

	return s.repository.SetReorderLevel(
		ctx,
		variantID,
		*request.ReorderLevel,
	)
}

func (s *Service) ListMovements(
	ctx context.Context,
	variantID string,
	limit int,
	offset int,
) (MovementListResult, error) {
	variantID = strings.TrimSpace(
		variantID,
	)

	if variantID == "" {
		return MovementListResult{},
			ErrInvalidInput
	}

	if limit <= 0 {
		limit = 50
	}

	if limit > 200 {
		limit = 200
	}

	if offset < 0 {
		offset = 0
	}

	return s.repository.ListMovements(
		ctx,
		variantID,
		limit,
		offset,
	)
}

func (r *Repository) List(
	ctx context.Context,
	limit int,
	offset int,
) (StockListResult, error) {
	const query = `
		SELECT
			v.id::text,
			v.sku,
			p.id::text,
			p.product_code,
			p.name,
			COALESCE(i.quantity_on_hand, 0),
			COALESCE(i.quantity_reserved, 0),
			COALESCE(i.reorder_level, 0),
			COALESCE(i.updated_at, v.updated_at)
		FROM product_variants v
		JOIN products p
			ON p.id = v.product_id
		LEFT JOIN inventory i
			ON i.variant_id = v.id
		ORDER BY
			p.name,
			v.sku
		LIMIT $1
		OFFSET $2
	`

	rows, err := r.db.Query(
		ctx,
		query,
		limit,
		offset,
	)
	if err != nil {
		return StockListResult{},
			fmt.Errorf(
				"list inventory: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]StockItem,
		0,
	)

	for rows.Next() {
		item, err := scanStock(
			rows,
		)
		if err != nil {
			return StockListResult{},
				fmt.Errorf(
					"scan inventory: %w",
					err,
				)
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return StockListResult{},
			fmt.Errorf(
				"iterate inventory: %w",
				err,
			)
	}

	return StockListResult{
		Items:  items,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (r *Repository) Get(
	ctx context.Context,
	variantID string,
) (StockItem, error) {
	const query = `
		SELECT
			v.id::text,
			v.sku,
			p.id::text,
			p.product_code,
			p.name,
			COALESCE(i.quantity_on_hand, 0),
			COALESCE(i.quantity_reserved, 0),
			COALESCE(i.reorder_level, 0),
			COALESCE(i.updated_at, v.updated_at)
		FROM product_variants v
		JOIN products p
			ON p.id = v.product_id
		LEFT JOIN inventory i
			ON i.variant_id = v.id
		WHERE v.id = $1::uuid
	`

	item, err := scanStock(
		r.db.QueryRow(
			ctx,
			query,
			variantID,
		),
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return StockItem{},
			ErrNotFound
	}

	if err != nil {
		return StockItem{},
			fmt.Errorf(
				"get inventory: %w",
				err,
			)
	}

	return item, nil
}

func (r *Repository) SetReorderLevel(
	ctx context.Context,
	variantID string,
	reorderLevel int,
) (StockItem, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return StockItem{},
			fmt.Errorf(
				"begin reorder update: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := ensureInventoryRow(
		ctx,
		tx,
		variantID,
	); err != nil {
		return StockItem{},
			err
	}

	if _, err := lockInventory(
		ctx,
		tx,
		variantID,
	); err != nil {
		return StockItem{},
			err
	}

	const query = `
		UPDATE inventory
		SET
			reorder_level = $2,
			updated_at = now()
		WHERE variant_id = $1::uuid
	`

	if _, err := tx.Exec(
		ctx,
		query,
		variantID,
		reorderLevel,
	); err != nil {
		return StockItem{},
			fmt.Errorf(
				"update reorder level: %w",
				err,
			)
	}

	if err := tx.Commit(ctx); err != nil {
		return StockItem{},
			fmt.Errorf(
				"commit reorder update: %w",
				err,
			)
	}

	return r.Get(
		ctx,
		variantID,
	)
}

func (r *Repository) ListMovements(
	ctx context.Context,
	variantID string,
	limit int,
	offset int,
) (MovementListResult, error) {
	if _, err := r.Get(
		ctx,
		variantID,
	); err != nil {
		return MovementListResult{},
			err
	}

	const query = `
		SELECT
			id::text,
			variant_id::text,
			COALESCE(reservation_id::text, ''),
			movement_type,
			quantity_on_hand_delta,
			quantity_reserved_delta,
			quantity_on_hand_after,
			quantity_reserved_after,
			COALESCE(reference_type, ''),
			COALESCE(reference_id, ''),
			COALESCE(reason, ''),
			COALESCE(note, ''),
			COALESCE(actor_type, ''),
			COALESCE(actor_id, ''),
			created_at
		FROM inventory_movements
		WHERE variant_id = $1::uuid
		ORDER BY
			created_at DESC,
			id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := r.db.Query(
		ctx,
		query,
		variantID,
		limit,
		offset,
	)
	if err != nil {
		return MovementListResult{},
			fmt.Errorf(
				"list inventory movements: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]Movement,
		0,
	)

	for rows.Next() {
		var movement Movement

		if err := rows.Scan(
			&movement.ID,
			&movement.VariantID,
			&movement.ReservationID,
			&movement.MovementType,
			&movement.QuantityOnHandDelta,
			&movement.QuantityReservedDelta,
			&movement.QuantityOnHandAfter,
			&movement.QuantityReservedAfter,
			&movement.ReferenceType,
			&movement.ReferenceID,
			&movement.Reason,
			&movement.Note,
			&movement.ActorType,
			&movement.ActorID,
			&movement.CreatedAt,
		); err != nil {
			return MovementListResult{},
				fmt.Errorf(
					"scan inventory movement: %w",
					err,
				)
		}

		items = append(
			items,
			movement,
		)
	}

	if err := rows.Err(); err != nil {
		return MovementListResult{},
			fmt.Errorf(
				"iterate inventory movements: %w",
				err,
			)
	}

	return MovementListResult{
		Items:  items,
		Limit:  limit,
		Offset: offset,
	}, nil
}
