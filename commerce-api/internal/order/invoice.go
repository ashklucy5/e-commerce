package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type InvoiceMerchant struct {
	Name string `json:"name,omitempty"`

	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`

	AddressLine1 string `json:"address_line1,omitempty"`
	AddressLine2 string `json:"address_line2,omitempty"`
	City         string `json:"city,omitempty"`
	Country      string `json:"country,omitempty"`
}

type InvoiceItem struct {
	OrderItemID string `json:"order_item_id"`
	VariantID   string `json:"variant_id"`

	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	Quantity int `json:"quantity"`

	UnitPriceAmount int64  `json:"unit_price_amount"`
	LineTotalAmount int64  `json:"line_total_amount"`
	Currency        string `json:"currency"`
}

type Invoice struct {
	ID string `json:"id"`

	InvoiceNumber string    `json:"invoice_number"`
	IssuedAt      time.Time `json:"issued_at"`

	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`

	Currency       string `json:"currency"`
	SubtotalAmount int64  `json:"subtotal_amount"`
	DiscountAmount int64  `json:"discount_amount"`
	ShippingAmount int64  `json:"shipping_amount"`
	TotalAmount    int64  `json:"total_amount"`

	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
	CustomerEmail string `json:"customer_email,omitempty"`

	ShippingAddressLine1 string `json:"shipping_address_line1"`
	ShippingAddressLine2 string `json:"shipping_address_line2,omitempty"`
	ShippingCity         string `json:"shipping_city"`
	ShippingArea         string `json:"shipping_area"`
	ShippingPostalCode   string `json:"shipping_postal_code,omitempty"`

	DeliveryMethod string `json:"delivery_method"`

	PaymentMethod        string `json:"payment_method"`
	PaymentStatusAtIssue string `json:"payment_status_at_issue"`

	Merchant InvoiceMerchant `json:"merchant"`

	Items []InvoiceItem `json:"items"`

	CustomerID string `json:"-"`
}

func (s *Service) Invoice(
	ctx context.Context,
	orderID string,
) (Invoice, error) {
	orderID =
		strings.TrimSpace(
			orderID,
		)

	if !uuidPattern.MatchString(
		orderID,
	) {
		return Invoice{},
			ErrInvalidOrderID
	}

	return s.repository.GetInvoiceByOrderID(
		ctx,
		orderID,
	)
}

func (s *Service) InvoiceForCustomer(
	ctx context.Context,
	customerID string,
	orderID string,
) (Invoice, error) {
	customerID =
		strings.TrimSpace(
			customerID,
		)

	invoice, err :=
		s.Invoice(
			ctx,
			orderID,
		)
	if err != nil {
		return Invoice{}, err
	}

	if invoice.CustomerID != "" &&
		invoice.CustomerID != customerID {
		return Invoice{},
			ErrOrderNotFound
	}

	return invoice, nil
}

func (r *Repository) GetInvoiceByOrderID(
	ctx context.Context,
	orderID string,
) (Invoice, error) {
	var result Invoice

	var merchantPayload []byte

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					i.id::text,
					i.invoice_number,
					i.issued_at,

					i.order_id::text,
					o.order_number,
					COALESCE(i.customer_id::text, ''),

					i.currency,
					i.subtotal_amount,
					i.discount_amount,
					i.shipping_amount,
					i.total_amount,

					i.customer_name,
					i.customer_phone,
					COALESCE(i.customer_email, ''),

					i.shipping_address_line1,
					COALESCE(i.shipping_address_line2, ''),
					i.shipping_city,
					i.shipping_area,
					COALESCE(i.shipping_postal_code, ''),

					i.delivery_method,
					i.payment_method,
					i.payment_status_at_issue,

					i.merchant_snapshot

				FROM invoices i
				JOIN orders o
					ON o.id = i.order_id

				WHERE
					i.order_id = $1::uuid
			`,
			orderID,
		).Scan(
			&result.ID,
			&result.InvoiceNumber,
			&result.IssuedAt,
			&result.OrderID,
			&result.OrderNumber,
			&result.CustomerID,
			&result.Currency,
			&result.SubtotalAmount,
			&result.DiscountAmount,
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
			&result.PaymentStatusAtIssue,
			&merchantPayload,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Invoice{},
			ErrInvoiceNotFound
	}

	if err != nil {
		return Invoice{},
			fmt.Errorf(
				"get invoice: %w",
				err,
			)
	}

	if len(merchantPayload) > 0 {
		if err :=
			json.Unmarshal(
				merchantPayload,
				&result.Merchant,
			); err != nil {

			return Invoice{},
				fmt.Errorf(
					"decode invoice merchant snapshot: %w",
					err,
				)
		}
	}

	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					order_item_id::text,
					variant_id::text,
					sku,
					product_name,
					quantity,
					unit_price_amount,
					line_total_amount,
					currency

				FROM invoice_items

				WHERE
					invoice_id = $1::uuid

				ORDER BY
					created_at,
					id
			`,
			result.ID,
		)
	if err != nil {
		return Invoice{},
			fmt.Errorf(
				"list invoice items: %w",
				err,
			)
	}

	defer rows.Close()

	result.Items =
		make(
			[]InvoiceItem,
			0,
		)

	for rows.Next() {
		var item InvoiceItem

		if err :=
			rows.Scan(
				&item.OrderItemID,
				&item.VariantID,
				&item.SKU,
				&item.ProductName,
				&item.Quantity,
				&item.UnitPriceAmount,
				&item.LineTotalAmount,
				&item.Currency,
			); err != nil {

			return Invoice{},
				fmt.Errorf(
					"scan invoice item: %w",
					err,
				)
		}

		result.Items =
			append(
				result.Items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return Invoice{},
			fmt.Errorf(
				"iterate invoice items: %w",
				err,
			)
	}

	return result, nil
}
