package promotion

import "time"

type EvaluateLine struct {
	VariantID       string
	Quantity        int
	UnitPriceAmount int64
}

type EvaluateInput struct {
	Code string

	SubtotalAmount int64
	Currency       string
	Lines          []EvaluateLine

	Now time.Time
}

type Result struct {
	Applied bool `json:"applied"`

	PromotionID   string `json:"promotion_id,omitempty"`
	PromotionCode string `json:"promotion_code,omitempty"`
	PromotionName string `json:"promotion_name,omitempty"`

	Scope        string `json:"scope,omitempty"`
	CampaignType string `json:"campaign_type,omitempty"`
	DiscountType string `json:"discount_type,omitempty"`

	DiscountAmount int64 `json:"discount_amount"`
}
