package returns

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (r *Repository) Begin(
	ctx context.Context,
) (pgx.Tx, error) {
	tx, err := r.db.Begin(
		ctx,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"begin return transaction: %w",
				err,
			)
	}

	return tx, nil
}

type orderSnapshot struct {
	ID     string
	Status string
}

type orderItemSnapshot struct {
	ID        string
	VariantID string

	SKU         string
	ProductName string

	Quantity int

	UnitPriceAmount int64
	Currency        string
}

func (r *Repository) LockOrderTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (orderSnapshot, error) {
	const query = `
		SELECT
			id::text,
			status
		FROM orders
		WHERE id = $1::uuid
		FOR UPDATE
	`

	var result orderSnapshot

	err := tx.QueryRow(
		ctx,
		query,
		orderID,
	).Scan(
		&result.ID,
		&result.Status,
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return orderSnapshot{},
			ErrOrderNotFound
	}

	if err != nil {
		return orderSnapshot{},
			fmt.Errorf(
				"lock return order: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ListOrderItemsTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) ([]orderItemSnapshot, error) {
	const query = `
		SELECT
			id::text,
			variant_id::text,
			sku,
			product_name,
			quantity,
			unit_price_amount,
			currency
		FROM order_items
		WHERE order_id = $1::uuid
		ORDER BY id
	`

	rows, err := tx.Query(
		ctx,
		query,
		orderID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list order items for return: %w",
				err,
			)
	}
	defer rows.Close()

	result := make(
		[]orderItemSnapshot,
		0,
	)

	for rows.Next() {
		var item orderItemSnapshot

		if err := rows.Scan(
			&item.ID,
			&item.VariantID,
			&item.SKU,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPriceAmount,
			&item.Currency,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan order item for return: %w",
					err,
				)
		}

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate order items for return: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ExistingReturnQuantitiesTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (map[string]int, error) {
	const query = `
		SELECT
			ri.order_item_id::text,
			COALESCE(SUM(ri.quantity), 0)::integer
		FROM return_items ri
		JOIN returns r
			ON r.id = ri.return_id
		WHERE
			r.order_id = $1::uuid
			AND r.status NOT IN (
				'rejected',
				'cancelled'
			)
		GROUP BY
			ri.order_item_id
	`

	rows, err := tx.Query(
		ctx,
		query,
		orderID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load existing return quantities: %w",
				err,
			)
	}
	defer rows.Close()

	result := make(
		map[string]int,
	)

	for rows.Next() {
		var orderItemID string
		var quantity int

		if err := rows.Scan(
			&orderItemID,
			&quantity,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan existing return quantity: %w",
					err,
				)
		}

		result[orderItemID] = quantity
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate existing return quantities: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) CreateTx(
	ctx context.Context,
	tx pgx.Tx,
	returnNumber string,
	orderID string,
	customerNote string,
	requestedBy string,
) (string, error) {
	const query = `
		INSERT INTO returns (
			return_number,
			order_id,
			status,
			customer_note,
			requested_by,
			requested_at,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2::uuid,
			'requested',
			NULLIF($3, ''),
			NULLIF($4, ''),
			now(),
			now(),
			now()
		)
		RETURNING id::text
	`

	var returnID string

	if err := tx.QueryRow(
		ctx,
		query,
		returnNumber,
		orderID,
		customerNote,
		requestedBy,
	).Scan(
		&returnID,
	); err != nil {
		return "",
			fmt.Errorf(
				"create return: %w",
				err,
			)
	}

	return returnID, nil
}

func (r *Repository) InsertItemsTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
	items []CreateItemRequest,
) error {
	const query = `
		INSERT INTO return_items (
			return_id,
			order_item_id,
			quantity,
			reason_code,
			reason_note,
			created_at,
			updated_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3,
			$4,
			NULLIF($5, ''),
			now(),
			now()
		)
	`

	for _, item := range items {
		if _, err := tx.Exec(
			ctx,
			query,
			returnID,
			item.OrderItemID,
			item.Quantity,
			item.ReasonCode,
			item.ReasonNote,
		); err != nil {
			return fmt.Errorf(
				"insert return item: %w",
				err,
			)
		}
	}

	return nil
}

func (r *Repository) LockByIDTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
) (Return, error) {
	result, err := scanReturn(
		tx.QueryRow(
			ctx,
			returnSelect+`
				WHERE id = $1::uuid
				FOR UPDATE
			`,
			returnID,
		),
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Return{},
			ErrReturnNotFound
	}

	if err != nil {
		return Return{},
			fmt.Errorf(
				"lock return: %w",
				err,
			)
	}

	items, err := getItems(
		ctx,
		tx,
		result.ID,
	)
	if err != nil {
		return Return{}, err
	}

	result.Items = items

	return result, nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	returnID string,
) (Return, error) {
	result, err := scanReturn(
		r.db.QueryRow(
			ctx,
			returnSelect+`
				WHERE id = $1::uuid
			`,
			returnID,
		),
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Return{},
			ErrReturnNotFound
	}

	if err != nil {
		return Return{},
			fmt.Errorf(
				"get return: %w",
				err,
			)
	}

	items, err := getItems(
		ctx,
		r.db,
		result.ID,
	)
	if err != nil {
		return Return{}, err
	}

	result.Items = items

	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

const returnSelect = `
	SELECT
		id::text,
		return_number,
		order_id::text,
		status,
		COALESCE(customer_note, ''),
		COALESCE(requested_by, ''),
		COALESCE(approved_by, ''),
		COALESCE(rejected_by, ''),
		COALESCE(rejection_reason, ''),
		requested_at,
		approved_at,
		rejected_at,
		received_at,
		inspected_at,
		completed_at,
		cancelled_at,
		created_at,
		updated_at
	FROM returns
`

func scanReturn(
	row rowScanner,
) (Return, error) {
	var result Return

	var approvedAt pgtype.Timestamptz
	var rejectedAt pgtype.Timestamptz
	var receivedAt pgtype.Timestamptz
	var inspectedAt pgtype.Timestamptz
	var completedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz

	err := row.Scan(
		&result.ID,
		&result.ReturnNumber,
		&result.OrderID,
		&result.Status,
		&result.CustomerNote,
		&result.RequestedBy,
		&result.ApprovedBy,
		&result.RejectedBy,
		&result.RejectionReason,
		&result.RequestedAt,
		&approvedAt,
		&rejectedAt,
		&receivedAt,
		&inspectedAt,
		&completedAt,
		&cancelledAt,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return Return{}, err
	}

	if approvedAt.Valid {
		value := approvedAt.Time
		result.ApprovedAt = &value
	}

	if rejectedAt.Valid {
		value := rejectedAt.Time
		result.RejectedAt = &value
	}

	if receivedAt.Valid {
		value := receivedAt.Time
		result.ReceivedAt = &value
	}

	if inspectedAt.Valid {
		value := inspectedAt.Time
		result.InspectedAt = &value
	}

	if completedAt.Valid {
		value := completedAt.Time
		result.CompletedAt = &value
	}

	if cancelledAt.Valid {
		value := cancelledAt.Time
		result.CancelledAt = &value
	}

	return result, nil
}

type itemQueryer interface {
	Query(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgx.Rows, error)
}

func getItems(
	ctx context.Context,
	queryer itemQueryer,
	returnID string,
) ([]Item, error) {
	const query = `
		SELECT
			ri.id::text,
			ri.return_id::text,
			ri.order_item_id::text,
			oi.variant_id::text,
			oi.sku,
			oi.product_name,
			ri.quantity,
			ri.reason_code,
			COALESCE(ri.reason_note, ''),
			ri.received_quantity,
			ri.restock_quantity,
			ri.inspection_status,
			COALESCE(ri.inspection_note, ''),
			oi.unit_price_amount,
			oi.currency,
			ri.created_at,
			ri.updated_at
		FROM return_items ri
		JOIN order_items oi
			ON oi.id = ri.order_item_id
		WHERE ri.return_id = $1::uuid
		ORDER BY
			ri.created_at,
			ri.id
	`

	rows, err := queryer.Query(
		ctx,
		query,
		returnID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load return items: %w",
				err,
			)
	}
	defer rows.Close()

	result := make(
		[]Item,
		0,
	)

	for rows.Next() {
		var item Item

		if err := rows.Scan(
			&item.ID,
			&item.ReturnID,
			&item.OrderItemID,
			&item.VariantID,
			&item.SKU,
			&item.ProductName,
			&item.Quantity,
			&item.ReasonCode,
			&item.ReasonNote,
			&item.ReceivedQuantity,
			&item.RestockQuantity,
			&item.InspectionStatus,
			&item.InspectionNote,
			&item.UnitPriceAmount,
			&item.Currency,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan return item: %w",
					err,
				)
		}

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate return items: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) MarkApprovedTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
	actorID string,
) error {
	const query = `
		UPDATE returns
		SET
			status = 'approved',
			approved_by = NULLIF($2, ''),
			approved_at = now(),
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'requested'
	`

	tag, err := tx.Exec(
		ctx,
		query,
		returnID,
		actorID,
	)
	if err != nil {
		return fmt.Errorf(
			"approve return: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidReturnTransition
	}

	return nil
}

func (r *Repository) MarkRejectedTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
	actorID string,
	reason string,
) error {
	const query = `
		UPDATE returns
		SET
			status = 'rejected',
			rejected_by = NULLIF($2, ''),
			rejection_reason = $3,
			rejected_at = now(),
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'requested'
	`

	tag, err := tx.Exec(
		ctx,
		query,
		returnID,
		actorID,
		reason,
	)
	if err != nil {
		return fmt.Errorf(
			"reject return: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidReturnTransition
	}

	return nil
}

func (r *Repository) MarkCancelledTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
) error {
	const query = `
		UPDATE returns
		SET
			status = 'cancelled',
			cancelled_at = now(),
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'requested'
	`

	tag, err := tx.Exec(
		ctx,
		query,
		returnID,
	)
	if err != nil {
		return fmt.Errorf(
			"cancel return: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidReturnTransition
	}

	return nil
}

func (r *Repository) UpdateReceivedItemsTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
	items []ReceiveItemRequest,
) error {
	const query = `
		UPDATE return_items
		SET
			received_quantity = $3,
			updated_at = now()
		WHERE
			return_id = $1::uuid
			AND order_item_id = $2::uuid
	`

	for _, item := range items {
		tag, err := tx.Exec(
			ctx,
			query,
			returnID,
			item.OrderItemID,
			item.ReceivedQuantity,
		)
		if err != nil {
			return fmt.Errorf(
				"update received return item: %w",
				err,
			)
		}

		if tag.RowsAffected() != 1 {
			return ErrReturnItemNotFound
		}
	}

	return nil
}

func (r *Repository) MarkReceivedTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
) error {
	const query = `
		UPDATE returns
		SET
			status = 'received',
			received_at = now(),
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'approved'
	`

	tag, err := tx.Exec(
		ctx,
		query,
		returnID,
	)
	if err != nil {
		return fmt.Errorf(
			"mark return received: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidReturnTransition
	}

	return nil
}

func (r *Repository) UpdateInspectedItemTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
	item InspectItemRequest,
) error {
	const query = `
		UPDATE return_items
		SET
			inspection_status = $3,
			inspection_note = NULLIF($4, ''),
			restock_quantity = $5,
			updated_at = now()
		WHERE
			return_id = $1::uuid
			AND order_item_id = $2::uuid
	`

	tag, err := tx.Exec(
		ctx,
		query,
		returnID,
		item.OrderItemID,
		item.InspectionStatus,
		item.InspectionNote,
		item.RestockQuantity,
	)
	if err != nil {
		return fmt.Errorf(
			"update inspected return item: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrReturnItemNotFound
	}

	return nil
}

func (r *Repository) MarkInspectedTx(
	ctx context.Context,
	tx pgx.Tx,
	returnID string,
) error {
	const query = `
		UPDATE returns
		SET
			status = 'inspected',
			inspected_at = now(),
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'received'
	`

	tag, err := tx.Exec(
		ctx,
		query,
		returnID,
	)
	if err != nil {
		return fmt.Errorf(
			"mark return inspected: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidReturnTransition
	}

	return nil
}
