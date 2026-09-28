package order

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
				"begin order transaction: %w",
				err,
			)
	}

	return tx, nil
}

type checkoutSnapshot struct {
	ID          string
	CheckoutKey string
	CartID      string
	CustomerID  string

	Status     string
	CartStatus string
	Currency   string

	SubtotalAmount int64
	DiscountAmount int64

	PromotionID   string
	PromotionCode string

	ShippingAmount int64
	TotalAmount    int64

	CustomerName  string
	CustomerPhone string
	CustomerEmail string

	ShippingAddressLine1 string
	ShippingAddressLine2 string
	ShippingCity         string
	ShippingArea         string
	ShippingPostalCode   string

	DeliveryMethod string
	PaymentMethod  string

	ExpiresAt time.Time
}

type checkoutItemSnapshot struct {
	VariantID   string
	SKU         string
	ProductName string

	Quantity             int
	MinimumOrderQuantity int

	UnitPriceAmount int64
	LineTotalAmount int64
	Currency        string

	CurrentMinimumOrderQuantity int
	CurrentUnitPriceAmount      int64
	CurrentCurrency             string
	VariantActive               bool
	ProductActive               bool
}

func (r *Repository) LockCheckoutForPlaceOrder(
	ctx context.Context,
	tx pgx.Tx,
	checkoutKey string,
) (checkoutSnapshot, error) {
	const query = `
		SELECT
			cs.id::text,
			cs.checkout_key,
			cs.cart_id::text,
			COALESCE(cs.customer_id::text, ''),
			cs.status,
			c.status,
			cs.currency,
			cs.subtotal_amount,
			cs.discount_amount,
			COALESCE(cs.promotion_id::text, ''),
			COALESCE(cs.promotion_code, ''),
			cs.shipping_amount,
			cs.total_amount,
			COALESCE(cs.customer_name, ''),
			COALESCE(cs.customer_phone, ''),
			COALESCE(cs.customer_email, ''),
			COALESCE(cs.shipping_address_line1, ''),
			COALESCE(cs.shipping_address_line2, ''),
			COALESCE(cs.shipping_city, ''),
			COALESCE(cs.shipping_area, ''),
			COALESCE(cs.shipping_postal_code, ''),
			COALESCE(cs.delivery_method, ''),
			COALESCE(cs.payment_method, ''),
			cs.expires_at
		FROM checkout_sessions cs
		JOIN carts c
			ON c.id = cs.cart_id
		WHERE cs.checkout_key = $1
		FOR UPDATE OF cs, c
	`

	var result checkoutSnapshot

	err :=
		tx.QueryRow(
			ctx,
			query,
			checkoutKey,
		).Scan(
			&result.ID,
			&result.CheckoutKey,
			&result.CartID,
			&result.CustomerID,
			&result.Status,
			&result.CartStatus,
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
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return checkoutSnapshot{},
			ErrCheckoutNotFound
	}

	if err != nil {
		return checkoutSnapshot{},
			fmt.Errorf(
				"lock checkout: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) AttachCheckoutCustomerTx(
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
			"attach customer to checkout before order: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrCheckoutNotFound
	}

	return nil
}

func (r *Repository) GetCheckoutItemsTx(
	ctx context.Context,
	tx pgx.Tx,
	checkoutID string,
) ([]checkoutItemSnapshot, error) {
	const query = `
		SELECT
			ci.variant_id::text,
			ci.sku,
			ci.product_name,
			ci.quantity,
			ci.minimum_order_quantity,
			ci.unit_price_amount,
			ci.line_total_amount,
			ci.currency,
			v.minimum_order_quantity,
			COALESCE(
				tier.unit_price_amount,
				v.price_amount
			),
			v.currency,
			v.is_active,
			p.status = 'active'
		FROM checkout_items ci
		JOIN product_variants v
			ON v.id = ci.variant_id
		JOIN products p
			ON p.id = v.product_id
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
		) tier
			ON true
		WHERE ci.checkout_id = $1::uuid
		ORDER BY
			ci.created_at,
			ci.id
	`

	rows, err :=
		tx.Query(
			ctx,
			query,
			checkoutID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load checkout items for order: %w",
				err,
			)
	}
	defer rows.Close()

	items :=
		make(
			[]checkoutItemSnapshot,
			0,
		)

	for rows.Next() {
		var item checkoutItemSnapshot

		if err :=
			rows.Scan(
				&item.VariantID,
				&item.SKU,
				&item.ProductName,
				&item.Quantity,
				&item.MinimumOrderQuantity,
				&item.UnitPriceAmount,
				&item.LineTotalAmount,
				&item.Currency,
				&item.CurrentMinimumOrderQuantity,
				&item.CurrentUnitPriceAmount,
				&item.CurrentCurrency,
				&item.VariantActive,
				&item.ProductActive,
			); err != nil {
			return nil,
				fmt.Errorf(
					"scan checkout item for order: %w",
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
				"iterate checkout items for order: %w",
				err,
			)
	}

	return items, nil
}

type createOrderInput struct {
	OrderNumber string
	Checkout    checkoutSnapshot

	Status        string
	PaymentStatus string

	PaymentDueAt *time.Time
	ConfirmedAt  *time.Time
}

func (r *Repository) CreateOrderTx(
	ctx context.Context,
	tx pgx.Tx,
	input createOrderInput,
) (string, error) {
	const query = `
		INSERT INTO orders (
			order_number,
			checkout_id,
			cart_id,
			customer_id,
			status,
			payment_status,
			payment_method,
			currency,
			subtotal_amount,
			discount_amount,
			promotion_id,
			promotion_code,
			shipping_amount,
			total_amount,
			customer_name,
			customer_phone,
			customer_email,
			shipping_address_line1,
			shipping_address_line2,
			shipping_city,
			shipping_area,
			shipping_postal_code,
			delivery_method,
			payment_due_at,
			confirmed_at,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2::uuid,
			$3::uuid,
			NULLIF($4, '')::uuid,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			NULLIF($11, '')::uuid,
			NULLIF($12, ''),
			$13,
			$14,
			$15,
			$16,
			NULLIF($17, ''),
			$18,
			NULLIF($19, ''),
			$20,
			$21,
			NULLIF($22, ''),
			$23,
			$24,
			$25,
			now(),
			now()
		)
		RETURNING id::text
	`

	checkout :=
		input.Checkout

	var orderID string

	err :=
		tx.QueryRow(
			ctx,
			query,
			input.OrderNumber,
			checkout.ID,
			checkout.CartID,
			checkout.CustomerID,
			input.Status,
			input.PaymentStatus,
			checkout.PaymentMethod,
			checkout.Currency,
			checkout.SubtotalAmount,
			checkout.DiscountAmount,
			checkout.PromotionID,
			checkout.PromotionCode,
			checkout.ShippingAmount,
			checkout.TotalAmount,
			checkout.CustomerName,
			checkout.CustomerPhone,
			checkout.CustomerEmail,
			checkout.ShippingAddressLine1,
			checkout.ShippingAddressLine2,
			checkout.ShippingCity,
			checkout.ShippingArea,
			checkout.ShippingPostalCode,
			checkout.DeliveryMethod,
			input.PaymentDueAt,
			input.ConfirmedAt,
		).Scan(
			&orderID,
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"create order: %w",
				err,
			)
	}

	return orderID, nil
}

func (r *Repository) InsertOrderItemsTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	items []checkoutItemSnapshot,
) error {
	const query = `
		INSERT INTO order_items (
			order_id,
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
	`

	for _, item := range items {
		if _, err :=
			tx.Exec(
				ctx,
				query,
				orderID,
				item.VariantID,
				item.SKU,
				item.ProductName,
				item.Quantity,
				item.MinimumOrderQuantity,
				item.UnitPriceAmount,
				item.LineTotalAmount,
				item.Currency,
			); err != nil {
			return fmt.Errorf(
				"insert order item: %w",
				err,
			)
		}
	}

	return nil
}

func (r *Repository) MarkCheckoutCompletedTx(
	ctx context.Context,
	tx pgx.Tx,
	checkoutID string,
	now time.Time,
) error {
	const query = `
		UPDATE checkout_sessions
		SET
			status = 'completed',
			completed_at = $2,
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'active'
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			checkoutID,
			now,
		)
	if err != nil {
		return fmt.Errorf(
			"complete checkout: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrCheckoutNotActive
	}

	return nil
}

func (r *Repository) MarkCheckoutExpiredTx(
	ctx context.Context,
	tx pgx.Tx,
	checkoutID string,
) error {
	const query = `
		UPDATE checkout_sessions
		SET
			status = 'expired',
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'active'
	`

	_, err :=
		tx.Exec(
			ctx,
			query,
			checkoutID,
		)
	if err != nil {
		return fmt.Errorf(
			"expire checkout: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) MarkCartConvertedTx(
	ctx context.Context,
	tx pgx.Tx,
	cartID string,
) error {
	const query = `
		UPDATE carts
		SET
			status = 'converted',
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'active'
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			cartID,
		)
	if err != nil {
		return fmt.Errorf(
			"convert cart: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrCartNotActive
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

const orderSelect = `
	SELECT
		id::text,
		order_number,
		order_type,
		COALESCE(checkout_id::text, ''),
		COALESCE(cart_id::text, ''),
		COALESCE(customer_id::text, ''),
		status,
		payment_status,
		payment_method,
		currency,
		subtotal_amount,
		discount_amount,
		COALESCE(promotion_id::text, ''),
		COALESCE(promotion_code, ''),
		shipping_amount,
		total_amount,
		customer_name,
		customer_phone,
		COALESCE(customer_email, ''),
		shipping_address_line1,
		COALESCE(shipping_address_line2, ''),
		shipping_city,
		shipping_area,
		COALESCE(shipping_postal_code, ''),
		delivery_method,
		payment_due_at,
		paid_at,
		confirmed_at,
		cancelled_at,
		COALESCE(cancellation_reason, ''),
		COALESCE(cancelled_by, ''),
		created_at,
		updated_at
	FROM orders
`

func scanOrder(
	row rowScanner,
) (Order, error) {
	var result Order

	var paymentDueAt pgtype.Timestamptz
	var paidAt pgtype.Timestamptz
	var confirmedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz

	err :=
		row.Scan(
			&result.ID,
			&result.OrderNumber,
			&result.OrderType,
			&result.CheckoutID,
			&result.CartID,
			&result.CustomerID,
			&result.Status,
			&result.PaymentStatus,
			&result.PaymentMethod,
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
			&paymentDueAt,
			&paidAt,
			&confirmedAt,
			&cancelledAt,
			&result.CancellationReason,
			&result.CancelledBy,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return Order{},
			err
	}

	if paymentDueAt.Valid {
		value :=
			paymentDueAt.Time

		result.PaymentDueAt =
			&value
	}

	if paidAt.Valid {
		value :=
			paidAt.Time

		result.PaidAt =
			&value
	}

	if confirmedAt.Valid {
		value :=
			confirmedAt.Time

		result.ConfirmedAt =
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

func (r *Repository) GetByID(
	ctx context.Context,
	orderID string,
) (Order, error) {
	result, err :=
		scanOrder(
			r.db.QueryRow(
				ctx,
				orderSelect+`
					WHERE id = $1::uuid
				`,
				orderID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Order{},
			ErrOrderNotFound
	}

	if err != nil {
		return Order{},
			fmt.Errorf(
				"get order: %w",
				err,
			)
	}

	items, err :=
		getOrderItems(
			ctx,
			r.db,
			result.ID,
		)
	if err != nil {
		return Order{},
			err
	}

	attachOrderItems(
		&result,
		items,
	)

	if err :=
		attachFulfillmentState(
			ctx,
			r.db,
			&result,
		); err != nil {
		return Order{},
			err
	}

	return result, nil
}

func (r *Repository) GetOrderByCheckoutIDTx(
	ctx context.Context,
	tx pgx.Tx,
	checkoutID string,
) (Order, error) {
	result, err :=
		scanOrder(
			tx.QueryRow(
				ctx,
				orderSelect+`
					WHERE checkout_id = $1::uuid
				`,
				checkoutID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Order{},
			ErrOrderNotFound
	}

	if err != nil {
		return Order{},
			fmt.Errorf(
				"get order by checkout: %w",
				err,
			)
	}

	items, err :=
		getOrderItems(
			ctx,
			tx,
			result.ID,
		)
	if err != nil {
		return Order{},
			err
	}

	attachOrderItems(
		&result,
		items,
	)

	if err :=
		attachFulfillmentState(
			ctx,
			tx,
			&result,
		); err != nil {
		return Order{},
			err
	}

	return result, nil
}

func (r *Repository) LockOrderByIDTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) (Order, error) {
	result, err :=
		scanOrder(
			tx.QueryRow(
				ctx,
				orderSelect+`
					WHERE id = $1::uuid
					FOR UPDATE
				`,
				orderID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Order{},
			ErrOrderNotFound
	}

	if err != nil {
		return Order{},
			fmt.Errorf(
				"lock order: %w",
				err,
			)
	}

	items, err :=
		getOrderItems(
			ctx,
			tx,
			result.ID,
		)
	if err != nil {
		return Order{},
			err
	}

	attachOrderItems(
		&result,
		items,
	)

	if err :=
		attachFulfillmentState(
			ctx,
			tx,
			&result,
		); err != nil {
		return Order{},
			err
	}

	return result, nil
}

func (r *Repository) MarkPaidTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
	paidAt time.Time,
	targetStatus string,
) error {
	const query = `
UPDATE orders
SET
status = $3::varchar(30),
payment_status = 'paid',
paid_at = $2,
confirmed_at =
CASE
WHEN $3::varchar(30) = 'confirmed'
THEN COALESCE(confirmed_at, $2)
ELSE confirmed_at
END,
updated_at = now()
WHERE
id = $1::uuid
AND status = 'pending_payment'
AND payment_status = 'pending'
`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			orderID,
			paidAt,
			targetStatus,
		)
	if err != nil {
		return fmt.Errorf(
			"mark order paid: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOrderNotPendingPayment
	}

	return nil
}

type duePendingOrder struct {
	ID        string
	OrderType string
}

func (r *Repository) ListDuePendingOrdersTx(
	ctx context.Context,
	tx pgx.Tx,
	now time.Time,
	limit int,
) ([]duePendingOrder, error) {
	const query = `
SELECT
id::text,
order_type
FROM orders
WHERE
status = 'pending_payment'
AND payment_status = 'pending'
AND payment_due_at IS NOT NULL
AND payment_due_at <= $1
ORDER BY
payment_due_at,
id
LIMIT $2
FOR UPDATE SKIP LOCKED
`

	rows, err :=
		tx.Query(
			ctx,
			query,
			now,
			limit,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load due pending orders: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]duePendingOrder,
			0,
		)

	for rows.Next() {
		var item duePendingOrder

		if err :=
			rows.Scan(
				&item.ID,
				&item.OrderType,
			); err != nil {
			return nil,
				fmt.Errorf(
					"scan due pending order: %w",
					err,
				)
		}

		result =
			append(
				result,
				item,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate due pending orders: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) MarkPaymentExpiredTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID string,
) error {
	const query = `
		UPDATE orders
		SET
			status = 'payment_expired',
			payment_status = 'expired',
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'pending_payment'
			AND payment_status = 'pending'
	`

	tag, err :=
		tx.Exec(
			ctx,
			query,
			orderID,
		)
	if err != nil {
		return fmt.Errorf(
			"mark order payment expired: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrOrderNotPendingPayment
	}

	return nil
}

type orderItemQueryer interface {
	Query(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgx.Rows, error)
}

func getOrderItems(
	ctx context.Context,
	queryer orderItemQueryer,
	orderID string,
) ([]Item, error) {
	const query = `
		SELECT
			id::text,
			order_id::text,
			variant_id::text,
			sku,
			product_name,
			quantity,
			minimum_order_quantity,
			unit_price_amount,
			line_total_amount,
			currency,
			created_at
		FROM order_items
		WHERE order_id = $1::uuid
		ORDER BY
			created_at,
			id
	`

	rows, err :=
		queryer.Query(
			ctx,
			query,
			orderID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"load order items: %w",
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
				&item.OrderID,
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
					"scan order item: %w",
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
				"iterate order items: %w",
				err,
			)
	}

	return items, nil
}

func attachOrderItems(
	order *Order,
	items []Item,
) {
	order.Items =
		items

	order.ItemCount =
		len(
			items,
		)

	totalQuantity :=
		0

	for _, item := range items {
		totalQuantity +=
			item.Quantity
	}

	order.QuantityTotal =
		totalQuantity
}
