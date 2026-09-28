package admin

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

func listRefundSummaries(
	ctx context.Context,
	querier adminReadQuerier,
	foreignKey string,
	foreignID string,
) (
	[]RefundSummary,
	error,
) {
	var query string

	switch foreignKey {
	case "order_id":
		query = `
			SELECT
				id::text,
				refund_number,
				COALESCE(return_id::text, ''),
				COALESCE(payment_id::text, ''),
				source_type,
				status,
				amount,
				currency,
				COALESCE(provider, ''),
				COALESCE(provider_refund_id, ''),
				requested_at,
				succeeded_at
			FROM refunds
			WHERE order_id = $1::uuid
			ORDER BY created_at DESC, id DESC
		`

	case "return_id":
		query = `
			SELECT
				id::text,
				refund_number,
				COALESCE(return_id::text, ''),
				COALESCE(payment_id::text, ''),
				source_type,
				status,
				amount,
				currency,
				COALESCE(provider, ''),
				COALESCE(provider_refund_id, ''),
				requested_at,
				succeeded_at
			FROM refunds
			WHERE return_id = $1::uuid
			ORDER BY created_at DESC, id DESC
		`

	case "payment_id":
		query = `
			SELECT
				id::text,
				refund_number,
				COALESCE(return_id::text, ''),
				COALESCE(payment_id::text, ''),
				source_type,
				status,
				amount,
				currency,
				COALESCE(provider, ''),
				COALESCE(provider_refund_id, ''),
				requested_at,
				succeeded_at
			FROM refunds
			WHERE payment_id = $1::uuid
			ORDER BY created_at DESC, id DESC
		`

	default:
		return nil,
			fmt.Errorf(
				"unsupported refund read key %q",
				foreignKey,
			)
	}

	rows, err :=
		querier.Query(
			ctx,
			query,
			foreignID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list admin refunds: %w",
				err,
			)
	}

	defer rows.Close()

	result :=
		make(
			[]RefundSummary,
			0,
		)

	for rows.Next() {
		var item RefundSummary

		var succeededAt pgtype.Timestamptz

		if err :=
			rows.Scan(
				&item.ID,
				&item.RefundNumber,
				&item.ReturnID,
				&item.PaymentID,
				&item.SourceType,
				&item.Status,
				&item.Amount,
				&item.Currency,
				&item.Provider,
				&item.ProviderRefundID,
				&item.RequestedAt,
				&succeededAt,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan admin refund: %w",
					err,
				)
		}

		item.SucceededAt =
			adminTimePointer(
				succeededAt,
			)

		result =
			append(
				result,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate admin refunds: %w",
				err,
			)
	}

	return result, nil
}
