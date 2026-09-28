package review

type CreateRequest struct {
	OrderItemID string `json:"order_item_id"`
	Rating      int    `json:"rating"`

	Title *string `json:"title"`
	Body  *string `json:"body"`
}

type UpdateRequest struct {
	Rating *int `json:"rating"`

	Title *string `json:"title"`
	Body  *string `json:"body"`
}
