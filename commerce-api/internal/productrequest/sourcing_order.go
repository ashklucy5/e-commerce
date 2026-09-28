package productrequest

import (
	"context"
	"fmt"

	"project.local/commerce-api/internal/order"
)

type PlaceSourcingOrderRequest struct {
	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
	CustomerEmail string `json:"customer_email,omitempty"`

	ShippingAddressLine1 string `json:"shipping_address_line1"`
	ShippingAddressLine2 string `json:"shipping_address_line2,omitempty"`
	ShippingCity         string `json:"shipping_city"`
	ShippingArea         string `json:"shipping_area"`
	ShippingPostalCode   string `json:"shipping_postal_code,omitempty"`

	PaymentMethod string `json:"payment_method"`
}

type PlaceSourcingOrderResult struct {
	Order   order.Order `json:"order"`
	Created bool        `json:"created"`
}

func (s *Service) PlaceSourcingOrder(
	ctx context.Context,
	customerID string,
	requestID string,
	request PlaceSourcingOrderRequest,
) (
	PlaceSourcingOrderResult,
	error,
) {
	if !validUUID(
		customerID,
	) ||
		!validUUID(
			requestID,
		) {

		return PlaceSourcingOrderResult{},
			ErrNotFound
	}

	if s.orderService == nil {
		return PlaceSourcingOrderResult{},
			fmt.Errorf(
				"sourcing order service is unavailable",
			)
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return PlaceSourcingOrderResult{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	current, err :=
		s.repository.
			LockCustomerRequestTx(
				ctx,
				tx,
				customerID,
				requestID,
			)
	if err != nil {
		return PlaceSourcingOrderResult{},
			err
	}

	confirmation, err :=
		s.repository.
			LockSourcingConfirmationTx(
				ctx,
				tx,
				requestID,
			)
	if err != nil {
		return PlaceSourcingOrderResult{},
			err
	}

	// ---------------------------------------------------------
	// Idempotent retry.
	//
	// Once the confirmation already owns an order, never create
	// another catalog product, variant or order.
	// ---------------------------------------------------------

	if confirmation.Status ==
		"order_created" {

		if confirmation.CreatedOrderID ==
			nil {

			return PlaceSourcingOrderResult{},
				ErrOfferConflict
		}

		orderID :=
			*confirmation.CreatedOrderID

		if err :=
			tx.Commit(
				ctx,
			); err != nil {

			return PlaceSourcingOrderResult{},
				fmt.Errorf(
					"commit existing sourcing order lookup: %w",
					err,
				)
		}

		existing, err :=
			s.orderService.Get(
				ctx,
				orderID,
			)
		if err != nil {
			return PlaceSourcingOrderResult{},
				err
		}

		return PlaceSourcingOrderResult{
			Order: existing,

			Created: false,
		}, nil
	}

	// A new order may only be created from the finalized immutable
	// confirmation while the sourcing request is still agreed.
	if confirmation.Status !=
		"confirmed" ||
		current.RequestStatus !=
			"agreed" {

		return PlaceSourcingOrderResult{},
			ErrOfferConflict
	}

	if current.ConversationStatus ==
		"closed" {

		return PlaceSourcingOrderResult{},
			ErrConversationClosed
	}

	// ---------------------------------------------------------
	// IMPORTANT:
	//
	// The HTTP request contributes only customer/contact/address
	// and payment choice.
	//
	// Product name, quantity, MOQ, unit price, shipping price,
	// currency and total are copied exclusively from the finalized
	// sourcing confirmation.
	// ---------------------------------------------------------

	creation, err :=
		s.orderService.
			CreateSourcingOrderTx(
				ctx,
				tx,
				order.SourcingOrderInput{
					ConfirmationID: confirmation.ID,

					CustomerID: customerID,

					ProductName: confirmation.
						AcceptedProductName,

					Quantity: confirmation.
						Quantity,

					MinimumOrderQuantity: confirmation.
						MinimumOrderQuantity,

					UnitPriceAmount: confirmation.
						UnitPriceSnapshot,

					ShippingAmount: confirmation.
						ShippingPriceSnapshot,

					TotalAmount: confirmation.
						TotalAmount,

					Currency: confirmation.
						Currency,

					CustomerName: request.CustomerName,

					CustomerPhone: request.CustomerPhone,

					CustomerEmail: request.CustomerEmail,

					ShippingAddressLine1: request.
						ShippingAddressLine1,

					ShippingAddressLine2: request.
						ShippingAddressLine2,

					ShippingCity: request.ShippingCity,

					ShippingArea: request.ShippingArea,

					ShippingPostalCode: request.
						ShippingPostalCode,

					PaymentMethod: request.PaymentMethod,
				},
			)
	if err != nil {
		return PlaceSourcingOrderResult{},
			err
	}

	if err :=
		s.repository.
			MarkSourcingOrderCreatedTx(
				ctx,
				tx,
				current.CaseID,
				requestID,
				confirmation.ID,
				creation.ProductID,
				creation.VariantID,
				creation.OrderID,
			); err != nil {

		return PlaceSourcingOrderResult{},
			err
	}

	if err :=
		s.repository.
			InsertCustomerEventTx(
				ctx,
				tx,
				current.CaseID,
				"product_sourcing_order_created",
				current.ConversationStatus,
				"closed",
				map[string]any{
					"source": "product_request_sourcing_order",

					"product_request_id": requestID,

					"confirmation_id": confirmation.ID,

					"product_id": creation.ProductID,

					"variant_id": creation.VariantID,

					"order_id": creation.OrderID,
				},
			); err != nil {

		return PlaceSourcingOrderResult{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return PlaceSourcingOrderResult{},
			fmt.Errorf(
				"commit sourcing order creation: %w",
				err,
			)
	}

	result, err :=
		s.orderService.Get(
			ctx,
			creation.OrderID,
		)
	if err != nil {
		return PlaceSourcingOrderResult{},
			err
	}

	return PlaceSourcingOrderResult{
		Order: result,

		Created: true,
	}, nil
}
