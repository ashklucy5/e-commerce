package order

type PlaceOrderResult struct {
	Order   Order
	Created bool
}

type CancelRequest struct {
	Reason string `json:"reason"`
}

type FulfillmentRequest struct {
	Status string `json:"status"`

	CourierName      string `json:"courier_name"`
	CourierReference string `json:"courier_reference"`
	TrackingNumber   string `json:"tracking_number"`
	TrackingURL      string `json:"tracking_url"`
}
