package promotion

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type StorefrontPromotion struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	Scope        string `json:"scope"`
	CampaignType string `json:"campaign_type"`
	DiscountType string `json:"discount_type"`

	PercentageBPS *int   `json:"percentage_bps,omitempty"`
	FixedAmount   *int64 `json:"fixed_amount,omitempty"`

	MinimumSubtotalAmount int64  `json:"minimum_subtotal_amount"`
	MaximumDiscountAmount *int64 `json:"maximum_discount_amount,omitempty"`

	Currency string `json:"currency"`

	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`
}

type storefrontStore interface {
	ListStorefrontActive(
		ctx context.Context,
		currency string,
		now time.Time,
	) ([]Promotion, error)
}

func (s *Service) ListStorefrontActive(
	ctx context.Context,
	currency string,
	now time.Time,
) ([]StorefrontPromotion, error) {
	currency =
		strings.ToUpper(
			strings.TrimSpace(
				currency,
			),
		)

	if currency == "" {
		currency = "BDT"
	}

	if len(currency) != 3 {
		return nil, ErrInvalidCurrency
	}

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	store, ok := s.store.(storefrontStore)
	if !ok {
		return nil,
			fmt.Errorf(
				"storefront promotion repository is unavailable",
			)
	}

	promotions, err :=
		store.ListStorefrontActive(
			ctx,
			currency,
			now,
		)
	if err != nil {
		return nil, err
	}

	result :=
		make(
			[]StorefrontPromotion,
			0,
			len(promotions),
		)

	for _, item := range promotions {
		result =
			append(
				result,
				StorefrontPromotion{
					ID: item.ID,

					Name: item.Name,

					Scope: normalizedScope(
						item.Scope,
					),

					CampaignType: normalizedCampaignType(
						item.CampaignType,
					),

					DiscountType: item.DiscountType,

					PercentageBPS: item.PercentageBPS,

					FixedAmount: item.FixedAmount,

					MinimumSubtotalAmount: item.MinimumSubtotalAmount,

					MaximumDiscountAmount: item.MaximumDiscountAmount,

					Currency: item.Currency,

					StartsAt: item.StartsAt,

					EndsAt: item.EndsAt,
				},
			)
	}

	return result, nil
}

func (r *Repository) ListStorefrontActive(
	ctx context.Context,
	currency string,
	now time.Time,
) ([]Promotion, error) {
	rows, err :=
		r.db.Query(
			ctx,
			promotionSelect+`
				WHERE
					code IS NULL
					AND scope = 'order'
					AND campaign_type = 'standard'
					AND status = 'active'
					AND currency = $1
					AND (
						starts_at IS NULL
						OR starts_at <= $2
					)
					AND (
						ends_at IS NULL
						OR ends_at > $2
					)
				ORDER BY
					ends_at ASC NULLS LAST,
					created_at DESC,
					id ASC
			`,
			currency,
			now,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list storefront promotions: %w",
				err,
			)
	}

	defer rows.Close()

	result := make([]Promotion, 0)

	for rows.Next() {
		item, err := scanPromotion(rows)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan storefront promotion: %w",
					err,
				)
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate storefront promotions: %w",
				err,
			)
	}

	return result, nil
}
