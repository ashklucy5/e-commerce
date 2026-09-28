package checkout

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	tx, err :=
		r.db.Begin(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"begin checkout transaction: %w",
				err,
			)
	}

	return tx, nil
}

func (r *Repository) LockCartSnapshot(
	ctx context.Context,
	tx pgx.Tx,
	cartKey string,
) (cartSnapshot, error) {
	var result cartSnapshot
	var expiresAt pgtype.Timestamptz

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					cart_key,
					status,
					currency,
					expires_at
				FROM carts
				WHERE cart_key = $1
				FOR UPDATE
			`,
			cartKey,
		).Scan(
			&result.ID,
			&result.CartKey,
			&result.Status,
			&result.Currency,
			&expiresAt,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return cartSnapshot{},
				ErrCartNotFound
		}

		return cartSnapshot{},
			fmt.Errorf(
				"lock checkout cart: %w",
				err,
			)
	}

	if expiresAt.Valid {
		value :=
			expiresAt.Time

		result.ExpiresAt =
			&value
	}

	rows, err :=
		tx.Query(
			ctx,
			`
				SELECT
					v.id::text,
					v.sku,
					p.name,
					ci.quantity,
					v.minimum_order_quantity,

					GREATEST(
						COALESCE(
							i.quantity_on_hand -
								i.quantity_reserved,
							0
						),
						0
					),

					COALESCE(
						price_tier.unit_price_amount,
						v.price_amount
					),

					v.currency,
					v.is_active,
					p.status = 'active'

				FROM cart_items ci

				JOIN product_variants v
					ON v.id = ci.variant_id

				JOIN products p
					ON p.id = v.product_id

				LEFT JOIN inventory i
					ON i.variant_id = v.id

				LEFT JOIN LATERAL (
					SELECT
						pt.unit_price_amount
					FROM product_variant_price_tiers pt
					WHERE
						pt.variant_id = v.id
						AND pt.min_quantity <= ci.quantity
					ORDER BY
						pt.min_quantity DESC
					LIMIT 1
				) price_tier
					ON true

				WHERE ci.cart_id = $1::uuid

				ORDER BY ci.created_at ASC
			`,
			result.ID,
		)
	if err != nil {
		return cartSnapshot{},
			fmt.Errorf(
				"load checkout cart items: %w",
				err,
			)
	}
	defer rows.Close()

	result.Items =
		make(
			[]cartSnapshotItem,
			0,
		)

	for rows.Next() {
		var item cartSnapshotItem

		if err :=
			rows.Scan(
				&item.VariantID,
				&item.SKU,
				&item.ProductName,
				&item.Quantity,
				&item.MinimumOrderQuantity,
				&item.AvailableQuantity,
				&item.UnitPriceAmount,
				&item.Currency,
				&item.VariantActive,
				&item.ProductActive,
			); err != nil {
			return cartSnapshot{},
				fmt.Errorf(
					"scan checkout cart item: %w",
					err,
				)
		}

		result.Items =
			append(
				result.Items,
				item,
			)
	}

	if err := rows.Err(); err != nil {
		return cartSnapshot{},
			fmt.Errorf(
				"iterate checkout cart items: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) GetActiveForCartTx(
	ctx context.Context,
	tx pgx.Tx,
	cartID string,
) (*Session, error) {
	row :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					checkout_key,
					cart_id::text,
					COALESCE(customer_id::text, ''),
					status,
					currency,
					subtotal_amount,
					discount_amount,
					COALESCE(promotion_id::text, ''),
					COALESCE(promotion_code, ''),
					shipping_amount,
					total_amount,

					COALESCE(customer_name, ''),
					COALESCE(customer_phone, ''),
					COALESCE(customer_email, ''),

					COALESCE(shipping_address_line1, ''),
					COALESCE(shipping_address_line2, ''),
					COALESCE(shipping_city, ''),
					COALESCE(shipping_area, ''),
					COALESCE(shipping_postal_code, ''),

					COALESCE(delivery_method, ''),
					COALESCE(payment_method, ''),

					expires_at,
					completed_at,
					cancelled_at,
					created_at,
					updated_at

				FROM checkout_sessions
				WHERE
					cart_id = $1::uuid
					AND status = 'active'
				LIMIT 1
				FOR UPDATE
			`,
			cartID,
		)

	result, err :=
		scanSession(
			row,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil, nil
		}

		return nil,
			fmt.Errorf(
				"get active checkout: %w",
				err,
			)
	}

	return &result, nil
}

func (r *Repository) CreateSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	checkoutKey string,
	cartID string,
	customerID string,
	currency string,
	subtotal int64,
	discountAmount int64,
	promotionID string,
	promotionCode string,
	expiresAt time.Time,
) (Session, error) {
	row :=
		tx.QueryRow(
			ctx,
			`
				INSERT INTO checkout_sessions (
					checkout_key,
					cart_id,
					customer_id,
					status,
					currency,
					subtotal_amount,
					discount_amount,
					promotion_id,
					promotion_code,
					shipping_amount,
					total_amount,
					expires_at,
					created_at,
					updated_at
				)
				VALUES (
					$1,
					$2::uuid,
					NULLIF($3, '')::uuid,
					'active',
					$4,
					$5,
					$6,
					NULLIF($7, '')::uuid,
					NULLIF($8, ''),
					0,
					$5::bigint - $6::bigint,
					$9,
					now(),
					now()
				)
				RETURNING
					id::text,
					checkout_key,
					cart_id::text,
					COALESCE(customer_id::text, ''),
					status,
					currency,
					subtotal_amount,
					discount_amount,
					COALESCE(promotion_id::text, ''),
					COALESCE(promotion_code, ''),
					shipping_amount,
					total_amount,

					COALESCE(customer_name, ''),
					COALESCE(customer_phone, ''),
					COALESCE(customer_email, ''),

					COALESCE(shipping_address_line1, ''),
					COALESCE(shipping_address_line2, ''),
					COALESCE(shipping_city, ''),
					COALESCE(shipping_area, ''),
					COALESCE(shipping_postal_code, ''),

					COALESCE(delivery_method, ''),
					COALESCE(payment_method, ''),

					expires_at,
					completed_at,
					cancelled_at,
					created_at,
					updated_at
			`,
			checkoutKey,
			cartID,
			customerID,
			currency,
			subtotal,
			discountAmount,
			promotionID,
			promotionCode,
			expiresAt,
		)

	result, err :=
		scanSession(
			row,
		)
	if err != nil {
		return Session{},
			fmt.Errorf(
				"create checkout session: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) RefreshSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	checkoutID string,
	currency string,
	subtotal int64,
	discountAmount int64,
	promotionID string,
	promotionCode string,
	expiresAt time.Time,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE checkout_sessions
				SET
					currency = $2,
					subtotal_amount = $3,
					discount_amount = $4,
					promotion_id =
						NULLIF($5, '')::uuid,
					promotion_code =
						NULLIF($6, ''),
					total_amount =
						$3::bigint -
							$4::bigint +
							shipping_amount,
					expires_at = $7,
					updated_at = now()
				WHERE
					id = $1::uuid
					AND status = 'active'
			`,
			checkoutID,
			currency,
			subtotal,
			discountAmount,
			promotionID,
			promotionCode,
			expiresAt,
		)
	if err != nil {
		return fmt.Errorf(
			"refresh checkout session: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) AttachCustomerTx(
	ctx context.Context,
	tx pgx.Tx,
	checkoutID string,
	customerID string,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE checkout_sessions
				SET
					customer_id = $2::uuid,
					updated_at = now()
				WHERE
					id = $1::uuid
					AND status = 'active'
					AND (
						customer_id IS NULL
						OR customer_id = $2::uuid
					)
			`,
			checkoutID,
			customerID,
		)
	if err != nil {
		return fmt.Errorf(
			"attach customer to checkout: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrCheckoutNotFound
	}

	return nil
}

func (r *Repository) ReplaceItemsTx(
	ctx context.Context,
	tx pgx.Tx,
	checkoutID string,
	items []Item,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				DELETE FROM checkout_items
				WHERE checkout_id = $1::uuid
			`,
			checkoutID,
		)
	if err != nil {
		return fmt.Errorf(
			"clear checkout items: %w",
			err,
		)
	}

	for _, item := range items {
		_, err :=
			tx.Exec(
				ctx,
				`
					INSERT INTO checkout_items (
						checkout_id,
						variant_id,
						sku,
						product_name,
						quantity,
						minimum_order_quantity,
						unit_price_amount,
						line_total_amount,
						currency,
						created_at
					)
					VALUES (
						$1::uuid,
						$2::uuid,
						$3,
						$4,
						$5,
						$6,
						$7,
						$8,
						$9,
						now()
					)
				`,
				checkoutID,
				item.VariantID,
				item.SKU,
				item.ProductName,
				item.Quantity,
				item.MinimumOrderQuantity,
				item.UnitPriceAmount,
				item.LineTotalAmount,
				item.Currency,
			)
		if err != nil {
			return fmt.Errorf(
				"insert checkout item %s: %w",
				item.SKU,
				err,
			)
		}
	}

	return nil
}

func (r *Repository) MarkExpiredTx(
	ctx context.Context,
	tx pgx.Tx,
	checkoutID string,
) error {
	_, err :=
		tx.Exec(
			ctx,
			`
				UPDATE checkout_sessions
				SET
					status = 'expired',
					updated_at = now()
				WHERE
					id = $1::uuid
					AND status = 'active'
			`,
			checkoutID,
		)
	if err != nil {
		return fmt.Errorf(
			"expire checkout session: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) MarkExpired(
	ctx context.Context,
	checkoutID string,
) error {
	_, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE checkout_sessions
				SET
					status = 'expired',
					updated_at = now()
				WHERE
					id = $1::uuid
					AND status = 'active'
			`,
			checkoutID,
		)
	if err != nil {
		return fmt.Errorf(
			"expire checkout session: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) MarkCancelled(
	ctx context.Context,
	checkoutID string,
) error {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE checkout_sessions
				SET
					status = 'cancelled',
					cancelled_at = now(),
					updated_at = now()
				WHERE
					id = $1::uuid
					AND status = 'active'
			`,
			checkoutID,
		)
	if err != nil {
		return fmt.Errorf(
			"cancel checkout session: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return ErrCheckoutNotActive
	}

	return nil
}

func (r *Repository) UpdateDetails(
	ctx context.Context,
	session Session,
) error {
	tag, err :=
		r.db.Exec(
			ctx,
			`
				UPDATE checkout_sessions
				SET
					customer_name =
						NULLIF($2, ''),

					customer_phone =
						NULLIF($3, ''),

					customer_email =
						NULLIF($4, ''),

					shipping_address_line1 =
						NULLIF($5, ''),

					shipping_address_line2 =
						NULLIF($6, ''),

					shipping_city =
						NULLIF($7, ''),

					shipping_area =
						NULLIF($8, ''),

					shipping_postal_code =
						NULLIF($9, ''),

					delivery_method =
						NULLIF($10, ''),

					payment_method =
						NULLIF($11, ''),

					promotion_id =
						NULLIF($12, '')::uuid,

					promotion_code =
						NULLIF($13, ''),

					discount_amount = $14,

					shipping_amount = $15,

					total_amount = $16,

					updated_at = now()

				WHERE
					id = $1::uuid
					AND status = 'active'
			`,
			session.ID,
			session.CustomerName,
			session.CustomerPhone,
			session.CustomerEmail,
			session.ShippingAddressLine1,
			session.ShippingAddressLine2,
			session.ShippingCity,
			session.ShippingArea,
			session.ShippingPostalCode,
			session.DeliveryMethod,
			session.PaymentMethod,
			session.PromotionID,
			session.PromotionCode,
			session.DiscountAmount,
			session.ShippingAmount,
			session.TotalAmount,
		)
	if err != nil {
		return fmt.Errorf(
			"update checkout details: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return ErrCheckoutNotActive
	}

	return nil
}

func (r *Repository) GetByKey(
	ctx context.Context,
	checkoutKey string,
) (Session, error) {
	row :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					checkout_key,
					cart_id::text,
					COALESCE(customer_id::text, ''),
					status,
					currency,
					subtotal_amount,
					discount_amount,
					COALESCE(promotion_id::text, ''),
					COALESCE(promotion_code, ''),
					shipping_amount,
					total_amount,

					COALESCE(customer_name, ''),
					COALESCE(customer_phone, ''),
					COALESCE(customer_email, ''),

					COALESCE(shipping_address_line1, ''),
					COALESCE(shipping_address_line2, ''),
					COALESCE(shipping_city, ''),
					COALESCE(shipping_area, ''),
					COALESCE(shipping_postal_code, ''),

					COALESCE(delivery_method, ''),
					COALESCE(payment_method, ''),

					expires_at,
					completed_at,
					cancelled_at,
					created_at,
					updated_at

				FROM checkout_sessions
				WHERE checkout_key = $1
			`,
			checkoutKey,
		)

	result, err :=
		scanSession(
			row,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Session{},
				ErrCheckoutNotFound
		}

		return Session{},
			fmt.Errorf(
				"get checkout session: %w",
				err,
			)
	}

	items, err :=
		r.getItems(
			ctx,
			result.ID,
		)
	if err != nil {
		return Session{},
			err
	}

	result.Items =
		items

	result.ItemCount =
		len(
			items,
		)

	for _, item := range items {
		result.QuantityTotal +=
			item.Quantity
	}

	return result, nil
}

func (r *Repository) getItems(
	ctx context.Context,
	checkoutID string,
) ([]Item, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					id::text,
					checkout_id::text,
					variant_id::text,
					sku,
					product_name,
					quantity,
					minimum_order_quantity,
					unit_price_amount,
					line_total_amount,
					currency,
					created_at

				FROM checkout_items

				WHERE checkout_id = $1::uuid

				ORDER BY created_at ASC
			`,
			checkoutID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"get checkout items: %w",
				err,
			)
	}
	defer rows.Close()

	items :=
		make(
			[]Item,
			0,
		)

	for rows.Next() {
		var item Item

		if err :=
			rows.Scan(
				&item.ID,
				&item.CheckoutID,
				&item.VariantID,
				&item.SKU,
				&item.ProductName,
				&item.Quantity,
				&item.MinimumOrderQuantity,
				&item.UnitPriceAmount,
				&item.LineTotalAmount,
				&item.Currency,
				&item.CreatedAt,
			); err != nil {
			return nil,
				fmt.Errorf(
					"scan checkout item: %w",
					err,
				)
		}

		items =
			append(
				items,
				item,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate checkout items: %w",
				err,
			)
	}

	return items, nil
}

func scanSession(
	row pgx.Row,
) (Session, error) {
	var result Session

	var completedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz

	err :=
		row.Scan(
			&result.ID,
			&result.CheckoutKey,
			&result.CartID,
			&result.CustomerID,
			&result.Status,
			&result.Currency,

			&result.SubtotalAmount,
			&result.DiscountAmount,
			&result.PromotionID,
			&result.PromotionCode,
			&result.ShippingAmount,
			&result.TotalAmount,

			&result.CustomerName,
			&result.CustomerPhone,
			&result.CustomerEmail,

			&result.ShippingAddressLine1,
			&result.ShippingAddressLine2,
			&result.ShippingCity,
			&result.ShippingArea,
			&result.ShippingPostalCode,

			&result.DeliveryMethod,
			&result.PaymentMethod,

			&result.ExpiresAt,
			&completedAt,
			&cancelledAt,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return Session{},
			err
	}

	if completedAt.Valid {
		value :=
			completedAt.Time

		result.CompletedAt =
			&value
	}

	if cancelledAt.Valid {
		value :=
			cancelledAt.Time

		result.CancelledAt =
			&value
	}

	return result, nil
}
