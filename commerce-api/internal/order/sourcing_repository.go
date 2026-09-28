package order

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const sourcingCategorySlug = "__internal-sourcing-orders"

type sourcingCatalogInput struct {
	ProductCode string
	ProductName string
	ProductSlug string

	SKU string

	MinimumOrderQuantity int
	UnitPriceAmount      int64
	Currency             string
}

type sourcingCatalogResult struct {
	ProductID string
	VariantID string
	SKU       string
}

func (r *Repository) CreateSourcingCatalogTx(
	ctx context.Context,
	tx pgx.Tx,
	input sourcingCatalogInput,
) (sourcingCatalogResult, error) {
	var categoryID string

	if err := tx.QueryRow(
		ctx,
		`
			INSERT INTO categories (
				name,
				slug,
				description,
				sort_order,
				is_active,
				created_at,
				updated_at
			)
			VALUES (
				'Internal Sourcing Orders',
				$1,
				'Internal category for negotiated sourcing-only products.',
				0,
				false,
				now(),
				now()
			)
			ON CONFLICT (slug)
			DO UPDATE SET
				is_active = false,
				updated_at = now()
			RETURNING id::text
		`,
		sourcingCategorySlug,
	).Scan(
		&categoryID,
	); err != nil {
		return sourcingCatalogResult{},
			fmt.Errorf(
				"ensure sourcing catalog category: %w",
				err,
			)
	}

	var productID string

	if err := tx.QueryRow(
		ctx,
		`
			INSERT INTO products (
				category_id,
				product_code,
				name,
				slug,
				short_description,
				description,
				status,
				is_featured,
				published_at,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				$4,
				'Negotiated sourcing-only product',
				'Internal sourcing product. Not published to the storefront.',
				'draft',
				false,
				NULL,
				now(),
				now()
			)
			RETURNING id::text
		`,
		categoryID,
		input.ProductCode,
		input.ProductName,
		input.ProductSlug,
	).Scan(
		&productID,
	); err != nil {
		return sourcingCatalogResult{},
			fmt.Errorf(
				"create sourcing product: %w",
				err,
			)
	}

	var variantID string

	if err := tx.QueryRow(
		ctx,
		`
			INSERT INTO product_variants (
				product_id,
				sku,
				minimum_order_quantity,
				order_increment,
				price_amount,
				compare_at_price_amount,
				cost_amount,
				currency,
				is_active,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				1,
				$4,
				NULL,
				NULL,
				$5,
				false,
				now(),
				now()
			)
			RETURNING id::text
		`,
		productID,
		input.SKU,
		input.MinimumOrderQuantity,
		input.UnitPriceAmount,
		input.Currency,
	).Scan(
		&variantID,
	); err != nil {
		return sourcingCatalogResult{},
			fmt.Errorf(
				"create sourcing variant: %w",
				err,
			)
	}

	if _, err := tx.Exec(
		ctx,
		`
			INSERT INTO inventory (
				variant_id,
				quantity_on_hand,
				quantity_reserved,
				reorder_level,
				updated_at
			)
			VALUES (
				$1::uuid,
				0,
				0,
				0,
				now()
			)
		`,
		variantID,
	); err != nil {
		return sourcingCatalogResult{},
			fmt.Errorf(
				"create sourcing inventory row: %w",
				err,
			)
	}

	return sourcingCatalogResult{
		ProductID: productID,
		VariantID: variantID,
		SKU:       input.SKU,
	}, nil
}

type createSourcingOrderInput struct {
	OrderNumber string
	CustomerID  string

	Status        string
	PaymentStatus string
	PaymentMethod string

	Currency string

	SubtotalAmount int64
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

	PaymentDueAt *time.Time
}

func (r *Repository) CreateSourcingOrderTx(
	ctx context.Context,
	tx pgx.Tx,
	input createSourcingOrderInput,
) (string, error) {
	var orderID string

	if err := tx.QueryRow(
		ctx,
		`
			INSERT INTO orders (
				order_number,
				order_type,
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
				'sourcing',
				NULL,
				NULL,
				$2::uuid,
				$3,
				$4,
				$5,
				$6,
				$7,
				0,
				NULL,
				NULL,
				$8,
				$9,
				$10,
				$11,
				NULLIF($12, ''),
				$13,
				NULLIF($14, ''),
				$15,
				$16,
				NULLIF($17, ''),
				$18,
				$19,
				NULL,
				now(),
				now()
			)
			RETURNING id::text
		`,
		input.OrderNumber,
		input.CustomerID,
		input.Status,
		input.PaymentStatus,
		input.PaymentMethod,
		input.Currency,
		input.SubtotalAmount,
		input.ShippingAmount,
		input.TotalAmount,
		input.CustomerName,
		input.CustomerPhone,
		input.CustomerEmail,
		input.ShippingAddressLine1,
		input.ShippingAddressLine2,
		input.ShippingCity,
		input.ShippingArea,
		input.ShippingPostalCode,
		input.DeliveryMethod,
		input.PaymentDueAt,
	).Scan(
		&orderID,
	); err != nil {
		return "",
			fmt.Errorf(
				"create sourcing order: %w",
				err,
			)
	}

	return orderID, nil
}
