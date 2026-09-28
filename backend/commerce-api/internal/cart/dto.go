package cart

type AddItemRequest struct {
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity"`
}
