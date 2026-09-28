package productrequest

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) LockSourcingConfirmationTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
) (
	SourcingConfirmation,
	error,
) {
	result, err :=
		scanSourcingConfirmation(
			tx.QueryRow(
				ctx,
				sourcingConfirmationSelect+`
WHERE
cf.request_id = $1::uuid
FOR UPDATE OF cf
`,
				requestID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return SourcingConfirmation{},
			ErrConfirmationNotFound
	}

	if err != nil {
		return SourcingConfirmation{},
			fmt.Errorf(
				"lock sourcing confirmation: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) MarkSourcingOrderCreatedTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	requestID string,
	confirmationID string,
	productID string,
	variantID string,
	orderID string,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
UPDATE product_sourcing_confirmations
SET
status = 'order_created',

created_product_id =
$3::uuid,

created_variant_id =
$4::uuid,

created_order_id =
$5::uuid,

updated_at = now()

WHERE
id = $1::uuid
AND request_id = $2::uuid
AND status = 'confirmed'
AND created_order_id IS NULL
`,
			confirmationID,
			requestID,
			productID,
			variantID,
			orderID,
		)
	if err != nil {
		return fmt.Errorf(
			"mark sourcing confirmation order created: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOfferConflict
	}

	tag, err =
		tx.Exec(
			ctx,
			`
UPDATE product_sourcing_requests
SET
status = 'converted_to_order',
status_reason = NULL,
updated_at = now()
WHERE
id = $1::uuid
AND status = 'agreed'
`,
			requestID,
		)
	if err != nil {
		return fmt.Errorf(
			"mark product request converted to order: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOfferConflict
	}

	tag, err =
		tx.Exec(
			ctx,
			`
UPDATE crm_cases
SET
status = 'closed',

resolved_at =
COALESCE(
resolved_at,
now()
),

closed_at =
COALESCE(
closed_at,
now()
),

updated_at = now()

WHERE
id = $1::uuid
`,
			caseID,
		)
	if err != nil {
		return fmt.Errorf(
			"close sourcing CRM case after order creation: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOfferConflict
	}

	return nil
}
