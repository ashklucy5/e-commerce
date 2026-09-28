package review

import "time"

type Review struct {
	ID          string `json:"id"`
	OrderItemID string `json:"order_item_id"`
	ProductID   string `json:"product_id"`
	VariantID   string `json:"variant_id"`

	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`

	Rating int     `json:"rating"`
	Title  *string `json:"title,omitempty"`
	Body   *string `json:"body,omitempty"`

	Status           string `json:"status"`
	VerifiedPurchase bool   `json:"verified_purchase"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PublicReview struct {
	ID        string `json:"id"`
	VariantID string `json:"variant_id"`
	SKU       string `json:"sku"`

	Rating int     `json:"rating"`
	Title  *string `json:"title,omitempty"`
	Body   *string `json:"body,omitempty"`

	ReviewerName     string `json:"reviewer_name"`
	VerifiedPurchase bool   `json:"verified_purchase"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RatingBreakdown struct {
	One   int64 `json:"1"`
	Two   int64 `json:"2"`
	Three int64 `json:"3"`
	Four  int64 `json:"4"`
	Five  int64 `json:"5"`
}

type Summary struct {
	ProductID string `json:"product_id"`

	TotalReviews          int64   `json:"total_reviews"`
	VerifiedPurchaseCount int64   `json:"verified_purchase_count"`
	AverageRating         float64 `json:"average_rating"`

	Breakdown RatingBreakdown `json:"breakdown"`
}
