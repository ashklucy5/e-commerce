package returns

type CreateRequest struct {
	CustomerNote string              `json:"customer_note"`
	Items        []CreateItemRequest `json:"items"`
}

type CreateItemRequest struct {
	OrderItemID string `json:"order_item_id"`
	Quantity    int    `json:"quantity"`
	ReasonCode  string `json:"reason_code"`
	ReasonNote  string `json:"reason_note"`
}

type RejectRequest struct {
	Reason string `json:"reason"`
}

type ReceiveRequest struct {
	Items []ReceiveItemRequest `json:"items"`
}

type ReceiveItemRequest struct {
	OrderItemID      string `json:"order_item_id"`
	ReceivedQuantity int    `json:"received_quantity"`
}

type InspectRequest struct {
	Items []InspectItemRequest `json:"items"`
}

type InspectItemRequest struct {
	OrderItemID string `json:"order_item_id"`

	InspectionStatus string `json:"inspection_status"`
	InspectionNote   string `json:"inspection_note"`

	RestockQuantity int `json:"restock_quantity"`
}
