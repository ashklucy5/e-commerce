package promotion

import "time"

const (
	DiscountTypePercentage = "percentage"
	DiscountTypeFixed      = "fixed"

	ScopeOrder   = "order"
	ScopeProduct = "product"

	CampaignTypeStandard  = "standard"
	CampaignTypeFlashSale = "flash_sale"

	StatusDraft    = "draft"
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

type Promotion struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`

	Scope        string `json:"scope"`
	CampaignType string `json:"campaign_type"`

	DiscountType string `json:"discount_type"`

	PercentageBPS *int   `json:"percentage_bps,omitempty"`
	FixedAmount   *int64 `json:"fixed_amount,omitempty"`

	MinimumSubtotalAmount int64  `json:"minimum_subtotal_amount"`
	MaximumDiscountAmount *int64 `json:"maximum_discount_amount,omitempty"`

	Currency string `json:"currency"`
	Status   string `json:"status"`

	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
