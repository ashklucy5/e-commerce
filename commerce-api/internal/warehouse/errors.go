package warehouse

import "errors"

var (
	ErrInvalidInput = errors.New(
		"invalid warehouse request",
	)

	ErrWarehouseNotFound = errors.New(
		"warehouse not found",
	)

	ErrWarehouseCodeExists = errors.New(
		"warehouse code already exists",
	)

	ErrWarehouseInactive = errors.New(
		"warehouse is inactive",
	)

	ErrOrderNotFound = errors.New(
		"order not found",
	)

	ErrOrderItemNotFound = errors.New(
		"order item not found",
	)

	ErrOrderNotFulfillable = errors.New(
		"order is not ready for warehouse fulfillment",
	)

	ErrAllocationExceeded = errors.New(
		"warehouse allocation exceeds the order item quantity",
	)

	ErrInboundShipmentRequired = errors.New(
		"china inbound fulfillment requires an inbound shipment",
	)

	ErrInboundShipmentNotFound = errors.New(
		"inbound shipment not found",
	)

	ErrInboundReferenceExists = errors.New(
		"inbound shipment reference already exists",
	)

	ErrInboundWarehouseMismatch = errors.New(
		"inbound shipment destination does not match the fulfillment warehouse",
	)

	ErrInboundSourceMismatch = errors.New(
		"inbound shipment is not a China-origin shipment",
	)

	ErrInboundTransitionNotAllowed = errors.New(
		"inbound shipment transition is not allowed",
	)

	ErrFulfillmentNotFound = errors.New(
		"warehouse fulfillment not found",
	)

	ErrFulfillmentTransitionNotAllowed = errors.New(
		"warehouse fulfillment transition is not allowed",
	)
)
