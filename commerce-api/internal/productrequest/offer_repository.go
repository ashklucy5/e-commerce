package productrequest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const sourcingOfferSelect = `
	SELECT
		o.id::text,
		o.request_id::text,
		o.status,
		o.product_name,
		o.description,
		COALESCE(o.attachments::text, ''),
		COALESCE(o.offered_specifications::text, ''),
		o.unit_price,
		o.shipping_price,
		o.currency,
		o.quoted_quantity,
		o.minimum_order_quantity,
		o.expires_at,
		o.sent_at,
		o.customer_responded_at,
		o.finalized_at,
		a.id::text,
		a.actor_code,
		a.actor_type,
		a.display_name,
		o.created_at,
		o.updated_at
	FROM product_sourcing_offers o
	JOIN support_actors a
		ON a.id = o.created_by
`

func scanSourcingOffer(
	row rowScanner,
) (
	SourcingOffer,
	error,
) {
	var result SourcingOffer

	var attachmentsText string
	var specificationsText string

	var quotedQuantity pgtype.Int4
	var minimumOrderQuantity pgtype.Int4

	var expiresAt pgtype.Timestamptz
	var sentAt pgtype.Timestamptz
	var respondedAt pgtype.Timestamptz
	var finalizedAt pgtype.Timestamptz

	var actor AdminActor

	err := row.Scan(
		&result.ID,
		&result.RequestID,
		&result.Status,
		&result.ProductName,
		&result.Description,
		&attachmentsText,
		&specificationsText,
		&result.UnitPrice,
		&result.ShippingPrice,
		&result.Currency,
		&quotedQuantity,
		&minimumOrderQuantity,
		&expiresAt,
		&sentAt,
		&respondedAt,
		&finalizedAt,
		&actor.ID,
		&actor.ActorCode,
		&actor.ActorType,
		&actor.DisplayName,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return SourcingOffer{}, err
	}

	if attachmentsText != "" {
		result.Attachments =
			json.RawMessage(
				attachmentsText,
			)
	}

	if specificationsText != "" {
		result.OfferedSpecifications =
			json.RawMessage(
				specificationsText,
			)
	}

	if quotedQuantity.Valid {
		value :=
			int(
				quotedQuantity.Int32,
			)

		result.QuotedQuantity =
			&value
	}

	if minimumOrderQuantity.Valid {
		value :=
			int(
				minimumOrderQuantity.Int32,
			)

		result.MinimumOrderQuantity =
			&value
	}

	if expiresAt.Valid {
		value :=
			expiresAt.Time

		result.ExpiresAt =
			&value
	}

	if sentAt.Valid {
		value :=
			sentAt.Time

		result.SentAt =
			&value
	}

	if respondedAt.Valid {
		value :=
			respondedAt.Time

		result.CustomerRespondedAt =
			&value
	}

	if finalizedAt.Valid {
		value :=
			finalizedAt.Time

		result.FinalizedAt =
			&value
	}

	result.CreatedBy =
		&actor

	return result,
		nil
}

func (r *Repository) ListAdminOffers(
	ctx context.Context,
	requestID string,
) (
	[]SourcingOffer,
	error,
) {
	if err :=
		r.requireProductRequest(
			ctx,
			requestID,
		); err != nil {

		return nil,
			err
	}

	rows, err :=
		r.db.Query(
			ctx,
			sourcingOfferSelect+`
				WHERE o.request_id = $1::uuid
				ORDER BY
					o.created_at DESC,
					o.id DESC
			`,
			requestID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list Admin sourcing offers: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]SourcingOffer,
			0,
		)

	for rows.Next() {
		item, err :=
			scanSourcingOffer(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan Admin sourcing offer: %w",
					err,
				)
		}

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
				"iterate Admin sourcing offers: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) ListCustomerOffers(
	ctx context.Context,
	customerID string,
	requestID string,
) (
	[]SourcingOffer,
	error,
) {
	if err :=
		r.requireCustomerProductRequest(
			ctx,
			customerID,
			requestID,
		); err != nil {

		return nil,
			err
	}

	rows, err :=
		r.db.Query(
			ctx,
			sourcingOfferSelect+`
				WHERE
					o.request_id = $1::uuid
					AND o.status <> 'draft'
				ORDER BY
					o.created_at DESC,
					o.id DESC
			`,
			requestID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list customer sourcing offers: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]SourcingOffer,
			0,
		)

	for rows.Next() {
		item, err :=
			scanSourcingOffer(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan customer sourcing offer: %w",
					err,
				)
		}

		item.CreatedBy =
			nil

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
				"iterate customer sourcing offers: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) GetCustomerOffer(
	ctx context.Context,
	customerID string,
	requestID string,
	offerID string,
) (
	SourcingOffer,
	error,
) {
	if err :=
		r.requireCustomerProductRequest(
			ctx,
			customerID,
			requestID,
		); err != nil {

		return SourcingOffer{},
			err
	}

	result, err :=
		scanSourcingOffer(
			r.db.QueryRow(
				ctx,
				sourcingOfferSelect+`
					WHERE
						o.id = $1::uuid
						AND o.request_id = $2::uuid
						AND o.status <> 'draft'
				`,
				offerID,
				requestID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return SourcingOffer{},
			ErrOfferNotFound
	}

	if err != nil {
		return SourcingOffer{},
			fmt.Errorf(
				"get customer sourcing offer: %w",
				err,
			)
	}

	result.CreatedBy =
		nil

	return result,
		nil
}

func (r *Repository) LockOfferTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	offerID string,
) (
	SourcingOffer,
	error,
) {
	result, err :=
		scanSourcingOffer(
			tx.QueryRow(
				ctx,
				sourcingOfferSelect+`
					WHERE
						o.id = $1::uuid
						AND o.request_id = $2::uuid
					FOR UPDATE OF o
				`,
				offerID,
				requestID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return SourcingOffer{},
			ErrOfferNotFound
	}

	if err != nil {
		return SourcingOffer{},
			fmt.Errorf(
				"lock sourcing offer: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) CreateOfferTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	actorID string,
	input offerMutation,
) (
	SourcingOffer,
	error,
) {
	var offerID string

	err :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO product_sourcing_offers (
					request_id,
					status,
					product_name,
					description,
					attachments,
					offered_specifications,
					unit_price,
					shipping_price,
					currency,
					created_by,
					quoted_quantity,
					minimum_order_quantity,
					expires_at,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					'draft',
					$2,
					$3,
					$4::jsonb,
					$5::jsonb,
					$6,
					$7,
					$8,
					$9::uuid,
					$10,
					$11,
					$12,
					now(),
					now()
				)
				RETURNING id::text
			`,
			requestID,
			input.ProductName,
			input.Description,
			jsonArgument(
				input.Attachments,
			),
			jsonArgument(
				input.OfferedSpecifications,
			),
			input.UnitPrice,
			input.ShippingPrice,
			input.Currency,
			actorID,
			input.QuotedQuantity,
			input.MinimumOrderQuantity,
			input.ExpiresAt,
		).Scan(
			&offerID,
		)

	if err != nil {
		return SourcingOffer{},
			fmt.Errorf(
				"create sourcing offer: %w",
				err,
			)
	}

	return r.LockOfferTx(
		ctx,
		tx,
		requestID,
		offerID,
	)
}

func (r *Repository) UpdateDraftOfferTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	offerID string,
	input offerMutation,
) (
	SourcingOffer,
	error,
) {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_offers
				SET
					product_name = $3,
					description = $4,
					attachments = $5::jsonb,
					offered_specifications = $6::jsonb,
					unit_price = $7,
					shipping_price = $8,
					currency = $9,
					quoted_quantity = $10,
					minimum_order_quantity = $11,
					expires_at = $12,
					updated_at = now()
				WHERE
					id = $1::uuid
					AND request_id = $2::uuid
					AND status = 'draft'
			`,
			offerID,
			requestID,
			input.ProductName,
			input.Description,
			jsonArgument(
				input.Attachments,
			),
			jsonArgument(
				input.OfferedSpecifications,
			),
			input.UnitPrice,
			input.ShippingPrice,
			input.Currency,
			input.QuotedQuantity,
			input.MinimumOrderQuantity,
			input.ExpiresAt,
		)

	if err != nil {
		return SourcingOffer{},
			fmt.Errorf(
				"update sourcing draft offer: %w",
				err,
			)
	}

	if tag.RowsAffected() != 1 {
		return SourcingOffer{},
			ErrOfferNotActionable
	}

	return r.LockOfferTx(
		ctx,
		tx,
		requestID,
		offerID,
	)
}

func (r *Repository) HasDraftOfferTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
) (
	bool,
	error,
) {
	var exists bool

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM product_sourcing_offers
					WHERE
						request_id = $1::uuid
						AND status = 'draft'
				)
			`,
			requestID,
		).Scan(
			&exists,
		)

	if err != nil {
		return false,
			fmt.Errorf(
				"check sourcing draft offer: %w",
				err,
			)
	}

	return exists,
		nil
}

func (r *Repository) HasAcceptedOrFinalizedOfferTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
) (
	bool,
	error,
) {
	var exists bool

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM product_sourcing_offers
					WHERE
						request_id = $1::uuid
						AND status IN (
							'customer_accepted',
							'finalized'
						)
				)
			`,
			requestID,
		).Scan(
			&exists,
		)

	if err != nil {
		return false,
			fmt.Errorf(
				"check accepted sourcing offer: %w",
				err,
			)
	}

	return exists,
		nil
}

func (r *Repository) ExpireRequestOffersTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_offers
				SET
					status = 'expired',
					updated_at = now()
				WHERE
					request_id = $1::uuid
					AND status = 'sent'
					AND expires_at IS NOT NULL
					AND expires_at <= now()
			`,
			requestID,
		)

	if err != nil {
		return fmt.Errorf(
			"expire sourcing offers: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) SupersedeSentOffersTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	exceptOfferID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_offers
				SET
					status = 'superseded',
					updated_at = now()
				WHERE
					request_id = $1::uuid
					AND id <> $2::uuid
					AND status = 'sent'
			`,
			requestID,
			exceptOfferID,
		)

	if err != nil {
		return fmt.Errorf(
			"supersede previous sourcing offers: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) SupersedeOtherOpenOffersTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	exceptOfferID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_offers
				SET
					status = 'superseded',
					updated_at = now()
				WHERE
					request_id = $1::uuid
					AND id <> $2::uuid
					AND status IN (
						'draft',
						'sent'
					)
			`,
			requestID,
			exceptOfferID,
		)

	if err != nil {
		return fmt.Errorf(
			"supersede open sourcing offers: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) SendOfferTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	offerID string,
) (
	SourcingOffer,
	error,
) {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_offers
				SET
					status = 'sent',
					sent_at = now(),
					customer_responded_at = NULL,
					updated_at = now()
				WHERE
					id = $1::uuid
					AND request_id = $2::uuid
					AND status = 'draft'
			`,
			offerID,
			requestID,
		)

	if err != nil {
		return SourcingOffer{},
			fmt.Errorf(
				"send sourcing offer: %w",
				err,
			)
	}

	if tag.RowsAffected() != 1 {
		return SourcingOffer{},
			ErrOfferNotActionable
	}

	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_requests
				SET
					status = 'negotiating',
					status_reason = NULL,
					updated_at = now()
				WHERE id = $1::uuid
			`,
			requestID,
		)

	if err != nil {
		return SourcingOffer{},
			fmt.Errorf(
				"mark sourcing request negotiating: %w",
				err,
			)
	}

	return r.LockOfferTx(
		ctx,
		tx,
		requestID,
		offerID,
	)
}

func (r *Repository) MarkOfferExpiredTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	offerID string,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_offers
				SET
					status = 'expired',
					updated_at = now()
				WHERE
					id = $1::uuid
					AND request_id = $2::uuid
					AND status = 'sent'
			`,
			offerID,
			requestID,
		)

	if err != nil {
		return fmt.Errorf(
			"mark sourcing offer expired: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOfferNotActionable
	}

	return nil
}

func (r *Repository) SetCustomerOfferResponseTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	offerID string,
	status string,
) (
	SourcingOffer,
	error,
) {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_offers
				SET
					status = $3::varchar,
					customer_responded_at = now(),
					updated_at = now()
				WHERE
					id = $1::uuid
					AND request_id = $2::uuid
					AND status = 'sent'
			`,
			offerID,
			requestID,
			status,
		)

	if err != nil {
		return SourcingOffer{},
			fmt.Errorf(
				"record customer sourcing offer response: %w",
				err,
			)
	}

	if tag.RowsAffected() != 1 {
		return SourcingOffer{},
			ErrOfferNotActionable
	}

	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_requests
				SET updated_at = now()
				WHERE id = $1::uuid
			`,
			requestID,
		)

	if err != nil {
		return SourcingOffer{},
			fmt.Errorf(
				"update sourcing request after offer response: %w",
				err,
			)
	}

	return r.LockOfferTx(
		ctx,
		tx,
		requestID,
		offerID,
	)
}

func (r *Repository) MarkCustomerOfferResponseCRMTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE crm_cases
				SET
					status = 'waiting_support',
					resolved_at = NULL,
					updated_at = now()
				WHERE id = $1::uuid
			`,
			caseID,
		)

	if err != nil {
		return fmt.Errorf(
			"update CRM after customer sourcing offer response: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ConfirmationExistsTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
) (
	bool,
	error,
) {
	var exists bool

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM product_sourcing_confirmations
					WHERE request_id = $1::uuid
				)
			`,
			requestID,
		).Scan(
			&exists,
		)

	if err != nil {
		return false,
			fmt.Errorf(
				"check sourcing confirmation: %w",
				err,
			)
	}

	return exists,
		nil
}

const sourcingConfirmationSelect = `
	SELECT
		cf.id::text,
		cf.request_id::text,
		cf.offer_id::text,
		cf.quantity,
		cf.minimum_order_quantity,
		cf.accepted_product_name,
		COALESCE(cf.accepted_specifications::text, ''),
		cf.unit_price_snapshot,
		cf.shipping_price_snapshot,
		cf.currency,
		cf.total_amount,
		cf.status,
		a.id::text,
		a.actor_code,
		a.actor_type,
		a.display_name,
		COALESCE(cf.created_product_id::text, ''),
		COALESCE(cf.created_variant_id::text, ''),
		COALESCE(cf.created_order_id::text, ''),
		cf.created_at,
		cf.updated_at
	FROM product_sourcing_confirmations cf
	JOIN support_actors a
		ON a.id = cf.finalized_by
`

func scanSourcingConfirmation(
	row rowScanner,
) (
	SourcingConfirmation,
	error,
) {
	var result SourcingConfirmation

	var specificationsText string

	var actor AdminActor

	var productID string
	var variantID string
	var orderID string

	err := row.Scan(
		&result.ID,
		&result.RequestID,
		&result.OfferID,
		&result.Quantity,
		&result.MinimumOrderQuantity,
		&result.AcceptedProductName,
		&specificationsText,
		&result.UnitPriceSnapshot,
		&result.ShippingPriceSnapshot,
		&result.Currency,
		&result.TotalAmount,
		&result.Status,
		&actor.ID,
		&actor.ActorCode,
		&actor.ActorType,
		&actor.DisplayName,
		&productID,
		&variantID,
		&orderID,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		return SourcingConfirmation{},
			err
	}

	if specificationsText != "" {
		result.AcceptedSpecifications =
			json.RawMessage(
				specificationsText,
			)
	}

	result.FinalizedBy =
		&actor

	if productID != "" {
		result.CreatedProductID =
			&productID
	}

	if variantID != "" {
		result.CreatedVariantID =
			&variantID
	}

	if orderID != "" {
		result.CreatedOrderID =
			&orderID
	}

	return result,
		nil
}

func (r *Repository) FinalizeOfferTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	offerID string,
	finalizedBy string,
	quantity int,
	minimumOrderQuantity int,
	productName string,
	specifications json.RawMessage,
	unitPrice int64,
	shippingPrice int64,
	currency string,
	totalAmount int64,
) (
	SourcingConfirmation,
	error,
) {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_offers
				SET
					status = 'finalized',
					finalized_at = now(),
					updated_at = now()
				WHERE
					id = $1::uuid
					AND request_id = $2::uuid
					AND status = 'customer_accepted'
			`,
			offerID,
			requestID,
		)

	if err != nil {
		return SourcingConfirmation{},
			fmt.Errorf(
				"finalize sourcing offer: %w",
				err,
			)
	}

	if tag.RowsAffected() != 1 {
		return SourcingConfirmation{},
			ErrOfferNotActionable
	}

	if err :=
		r.SupersedeOtherOpenOffersTx(
			ctx,
			tx,
			requestID,
			offerID,
		); err != nil {

		return SourcingConfirmation{},
			err
	}

	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_requests
				SET
					status = 'agreed',
					status_reason = NULL,
					updated_at = now()
				WHERE id = $1::uuid
			`,
			requestID,
		)

	if err != nil {
		return SourcingConfirmation{},
			fmt.Errorf(
				"mark sourcing request agreed: %w",
				err,
			)
	}

	var confirmationID string

	err =
		tx.QueryRow(
			ctx,
			`
				INSERT INTO product_sourcing_confirmations (
					request_id,
					offer_id,
					quantity,
					minimum_order_quantity,
					accepted_product_name,
					accepted_specifications,
					unit_price_snapshot,
					shipping_price_snapshot,
					currency,
					total_amount,
					status,
					finalized_by,
					created_at,
					updated_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3,
					$4,
					$5,
					$6::jsonb,
					$7,
					$8,
					$9,
					$10,
					'confirmed',
					$11::uuid,
					now(),
					now()
				)
				RETURNING id::text
			`,
			requestID,
			offerID,
			quantity,
			minimumOrderQuantity,
			productName,
			jsonArgument(
				specifications,
			),
			unitPrice,
			shippingPrice,
			currency,
			totalAmount,
			finalizedBy,
		).Scan(
			&confirmationID,
		)

	if err != nil {
		return SourcingConfirmation{},
			fmt.Errorf(
				"create sourcing confirmation: %w",
				err,
			)
	}

	result, err :=
		scanSourcingConfirmation(
			tx.QueryRow(
				ctx,
				sourcingConfirmationSelect+`
					WHERE
						cf.id = $1::uuid
						AND cf.request_id = $2::uuid
				`,
				confirmationID,
				requestID,
			),
		)

	if err != nil {
		return SourcingConfirmation{},
			fmt.Errorf(
				"load finalized sourcing confirmation: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) GetAdminConfirmation(
	ctx context.Context,
	requestID string,
) (
	SourcingConfirmation,
	error,
) {
	result, err :=
		scanSourcingConfirmation(
			r.db.QueryRow(
				ctx,
				sourcingConfirmationSelect+`
					WHERE cf.request_id = $1::uuid
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
				"get Admin sourcing confirmation: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) GetCustomerConfirmation(
	ctx context.Context,
	customerID string,
	requestID string,
) (
	SourcingConfirmation,
	error,
) {
	if err :=
		r.requireCustomerProductRequest(
			ctx,
			customerID,
			requestID,
		); err != nil {

		return SourcingConfirmation{},
			err
	}

	result, err :=
		scanSourcingConfirmation(
			r.db.QueryRow(
				ctx,
				sourcingConfirmationSelect+`
					WHERE cf.request_id = $1::uuid
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
				"get customer sourcing confirmation: %w",
				err,
			)
	}

	result.FinalizedBy =
		nil

	return result,
		nil
}

func (r *Repository) requireProductRequest(
	ctx context.Context,
	requestID string,
) error {
	var exists bool

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM product_sourcing_requests
					WHERE id = $1::uuid
				)
			`,
			requestID,
		).Scan(
			&exists,
		)

	if err != nil {
		return fmt.Errorf(
			"verify product request: %w",
			err,
		)
	}

	if !exists {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) requireCustomerProductRequest(
	ctx context.Context,
	customerID string,
	requestID string,
) error {
	var exists bool

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM product_sourcing_requests r
					JOIN crm_cases c
						ON c.id = r.case_id
					WHERE
						r.id = $1::uuid
						AND c.customer_id = $2::uuid
				)
			`,
			requestID,
			customerID,
		).Scan(
			&exists,
		)

	if err != nil {
		return fmt.Errorf(
			"verify customer product request: %w",
			err,
		)
	}

	if !exists {
		return ErrNotFound
	}

	return nil
}
