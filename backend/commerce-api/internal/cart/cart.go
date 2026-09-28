package cart

import "time"

type Cart struct {
	ID        string     `json:"id"`
	CartKey   string     `json:"cart_key"`
	Status    string     `json:"status"`
	Currency  string     `json:"currency"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	Items  []Item `json:"items"`
	Totals Totals `json:"totals"`
}

type Totals struct {
	Currency       string `json:"currency"`
	ItemCount      int    `json:"item_count"`
	QuantityTotal  int    `json:"quantity_total"`
	SubtotalAmount int64  `json:"subtotal_amount"`
}
