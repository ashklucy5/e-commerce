package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
)

func (s *Service) ListReturns(
	ctx context.Context,
	params platformpagination.Params,
	filter ReturnReadFilter,
) (
	ReturnListResult,
	error,
) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	rows, err := s.db.Query(
		queryCtx,
		`
			SELECT
				r.id::text,
				r.return_number,
				r.order_id::text,
				o.order_number,
				r.status,
				o.customer_name,
				o.customer_phone,
				r.requested_at,
				r.updated_at,
				COUNT(*) OVER()::bigint
			FROM returns r
			JOIN orders o
				ON o.id = r.order_id
			WHERE
				($1 = '' OR r.status = $1)
				AND (
					$2 = ''
					OR r.return_number = upper($2)
					OR o.order_number = upper($2)
					OR o.customer_phone = $2
				)
			ORDER BY r.created_at DESC, r.id DESC
			LIMIT $3 OFFSET $4
		`,
		filter.Status,
		filter.Query,
		params.Limit,
		params.Offset(),
	)
	if err != nil {
		return ReturnListResult{},
			fmt.Errorf(
				"list admin returns: %w",
				err,
			)
	}
	defer rows.Close()

	items := make(
		[]ReturnListItem,
		0,
		params.Limit,
	)

	var total int64

	for rows.Next() {
		var item ReturnListItem

		if err := rows.Scan(
			&item.ID,
			&item.ReturnNumber,
			&item.OrderID,
			&item.OrderNumber,
			&item.Status,
			&item.CustomerName,
			&item.CustomerPhone,
			&item.RequestedAt,
			&item.UpdatedAt,
			&total,
		); err != nil {
			return ReturnListResult{},
				fmt.Errorf(
					"scan admin return: %w",
					err,
				)
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return ReturnListResult{},
			fmt.Errorf(
				"iterate admin returns: %w",
				err,
			)
	}

	return ReturnListResult{
		Items: items,
		Meta: platformpagination.NewMeta(
			params,
			total,
		),
	}, nil
}

func (s *Service) GetReturn(
	ctx context.Context,
	returnID string,
) (
	ReturnDetail,
	error,
) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	var result ReturnDetail

	err := platformdatabase.WithinTxOptions(
		queryCtx,
		s.db,
		pgx.TxOptions{
			IsoLevel:   pgx.RepeatableRead,
			AccessMode: pgx.ReadOnly,
		},
		func(
			ctx context.Context,
			tx pgx.Tx,
		) error {
			if err := scanReturnDetail(
				ctx,
				tx,
				returnID,
				&result,
			); err != nil {
				return err
			}

			items, err := listReturnItemsRead(
				ctx,
				tx,
				returnID,
			)
			if err != nil {
				return err
			}

			refunds, err := listRefundSummaries(
				ctx,
				tx,
				"return_id",
				returnID,
			)
			if err != nil {
				return err
			}

			result.Items = items
			result.Refunds = refunds

			return nil
		},
	)
	if err != nil {
		return ReturnDetail{}, err
	}

	return result, nil
}

func scanReturnDetail(
	ctx context.Context,
	querier adminReadQuerier,
	returnID string,
	result *ReturnDetail,
) error {
	var approvedAt pgtype.Timestamptz
	var rejectedAt pgtype.Timestamptz
	var receivedAt pgtype.Timestamptz
	var inspectedAt pgtype.Timestamptz
	var completedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz

	err := querier.QueryRow(
		ctx,
		`
			SELECT
				r.id::text,
				r.return_number,
				r.order_id::text,
				o.order_number,
				r.status,
				o.customer_name,
				o.customer_phone,
				r.requested_at,
				r.updated_at,
				COALESCE(r.customer_note, ''),
				COALESCE(r.requested_by, ''),
				COALESCE(r.approved_by, ''),
				COALESCE(r.rejected_by, ''),
				COALESCE(r.rejection_reason, ''),
				r.approved_at,
				r.rejected_at,
				r.received_at,
				r.inspected_at,
				r.completed_at,
				r.cancelled_at
			FROM returns r
			JOIN orders o
				ON o.id = r.order_id
			WHERE r.id = $1::uuid
		`,
		returnID,
	).Scan(
		&result.ID,
		&result.ReturnNumber,
		&result.OrderID,
		&result.OrderNumber,
		&result.Status,
		&result.CustomerName,
		&result.CustomerPhone,
		&result.RequestedAt,
		&result.UpdatedAt,
		&result.CustomerNote,
		&result.RequestedBy,
		&result.ApprovedBy,
		&result.RejectedBy,
		&result.RejectionReason,
		&approvedAt,
		&rejectedAt,
		&receivedAt,
		&inspectedAt,
		&completedAt,
		&cancelledAt,
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrAdminReturnNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"get admin return: %w",
			err,
		)
	}

	result.ApprovedAt = adminTimePointer(approvedAt)
	result.RejectedAt = adminTimePointer(rejectedAt)
	result.ReceivedAt = adminTimePointer(receivedAt)
	result.InspectedAt = adminTimePointer(inspectedAt)
	result.CompletedAt = adminTimePointer(completedAt)
	result.CancelledAt = adminTimePointer(cancelledAt)

	return nil
}

func listReturnItemsRead(
	ctx context.Context,
	querier adminReadQuerier,
	returnID string,
) (
	[]ReturnItemRead,
	error,
) {
	rows, err := querier.Query(
		ctx,
		`
			SELECT
				ri.id::text,
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
				oi.currency
			FROM return_items ri
			JOIN order_items oi
				ON oi.id = ri.order_item_id
			WHERE ri.return_id = $1::uuid
			ORDER BY ri.created_at, ri.id
		`,
		returnID,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list admin return items: %w",
				err,
			)
	}
	defer rows.Close()

	result := make(
		[]ReturnItemRead,
		0,
	)

	for rows.Next() {
		var item ReturnItemRead

		if err := rows.Scan(
			&item.ID,
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
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan admin return item: %w",
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
				"iterate admin return items: %w",
				err,
			)
	}

	return result, nil
}
