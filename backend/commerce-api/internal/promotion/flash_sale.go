package promotion

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type StorefrontFlashSale struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	Scope        string `json:"scope"`
	CampaignType string `json:"campaign_type"`
	DiscountType string `json:"discount_type"`

	PercentageBPS *int   `json:"percentage_bps,omitempty"`
	FixedAmount   *int64 `json:"fixed_amount,omitempty"`

	MaximumDiscountAmount *int64 `json:"maximum_discount_amount,omitempty"`

	Currency string `json:"currency"`

	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`

	Targets []StorefrontFlashSaleTarget `json:"targets"`
}

type StorefrontFlashSaleTarget struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	ProductSlug string `json:"product_slug"`

	VariantID string `json:"variant_id"`
	SKU       string `json:"sku"`

	PriceAmount          int64  `json:"price_amount"`
	CompareAtPriceAmount *int64 `json:"compare_at_price_amount,omitempty"`

	EffectivePriceAmount int64 `json:"effective_price_amount"`
	DiscountAmount       int64 `json:"discount_amount"`

	Currency string `json:"currency"`

	PrimaryImageURL string `json:"primary_image_url,omitempty"`
}

type flashSaleTargetRecord struct {
	Promotion Promotion

	ProductID   string
	ProductName string
	ProductSlug string

	VariantID string
	SKU       string

	PriceAmount          int64
	CompareAtPriceAmount *int64
	Currency             string
	PrimaryImageURL      string
}

type flashSaleStore interface {
	ListActiveFlashSaleTargets(
		ctx context.Context,
		currency string,
		now time.Time,
	) ([]flashSaleTargetRecord, error)
}

func (s *Service) ListActiveFlashSales(
	ctx context.Context,
	currency string,
	now time.Time,
) ([]StorefrontFlashSale, error) {
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

	store, ok := s.store.(flashSaleStore)
	if !ok {
		return nil,
			fmt.Errorf(
				"flash sale repository is unavailable",
			)
	}

	records, err :=
		store.ListActiveFlashSaleTargets(
			ctx,
			currency,
			now,
		)
	if err != nil {
		return nil, err
	}

	result := make([]StorefrontFlashSale, 0)
	campaignIndex := make(map[string]int)
	variantCampaign := make(map[string]string)

	for _, record := range records {
		if previousPromotionID, exists := variantCampaign[record.VariantID]; exists && previousPromotionID != record.Promotion.ID {
			return nil,
				fmt.Errorf(
					"active flash sale target conflict for variant %s between promotions %s and %s",
					record.VariantID,
					previousPromotionID,
					record.Promotion.ID,
				)
		}

		variantCampaign[record.VariantID] = record.Promotion.ID

		index, exists := campaignIndex[record.Promotion.ID]

		if !exists {
			if record.Promotion.StartsAt == nil ||
				record.Promotion.EndsAt == nil {
				return nil, ErrInvalidPromotion
			}

			campaign := StorefrontFlashSale{
				ID:                    record.Promotion.ID,
				Name:                  record.Promotion.Name,
				Scope:                 ScopeProduct,
				CampaignType:          CampaignTypeFlashSale,
				DiscountType:          record.Promotion.DiscountType,
				PercentageBPS:         record.Promotion.PercentageBPS,
				FixedAmount:           record.Promotion.FixedAmount,
				MaximumDiscountAmount: record.Promotion.MaximumDiscountAmount,
				Currency:              record.Promotion.Currency,
				StartsAt:              record.Promotion.StartsAt.UTC(),
				EndsAt:                record.Promotion.EndsAt.UTC(),
				Targets:               make([]StorefrontFlashSaleTarget, 0),
			}

			result = append(result, campaign)
			index = len(result) - 1
			campaignIndex[record.Promotion.ID] = index
		}

		unitDiscount, err :=
			calculateProductUnitDiscount(
				record.Promotion,
				record.PriceAmount,
			)
		if err != nil {
			return nil, err
		}

		if record.Promotion.MaximumDiscountAmount != nil &&
			unitDiscount > *record.Promotion.MaximumDiscountAmount {
			unitDiscount = *record.Promotion.MaximumDiscountAmount
		}

		effectivePrice := record.PriceAmount - unitDiscount
		if effectivePrice < 0 {
			effectivePrice = 0
		}

		result[index].Targets = append(
			result[index].Targets,
			StorefrontFlashSaleTarget{
				ProductID:            record.ProductID,
				ProductName:          record.ProductName,
				ProductSlug:          record.ProductSlug,
				VariantID:            record.VariantID,
				SKU:                  record.SKU,
				PriceAmount:          record.PriceAmount,
				CompareAtPriceAmount: record.CompareAtPriceAmount,
				EffectivePriceAmount: effectivePrice,
				DiscountAmount:       unitDiscount,
				Currency:             record.Currency,
				PrimaryImageURL:      record.PrimaryImageURL,
			},
		)
	}

	return result, nil
}

func (r *Repository) ListActiveFlashSaleTargets(
	ctx context.Context,
	currency string,
	now time.Time,
) ([]flashSaleTargetRecord, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					promo.id::text,
					promo.name,
					COALESCE(promo.code, ''),
					promo.scope,
					promo.campaign_type,
					promo.discount_type,
					promo.percentage_bps,
					promo.fixed_amount,
					promo.minimum_subtotal_amount,
					promo.maximum_discount_amount,
					promo.currency,
					promo.status,
					promo.starts_at,
					promo.ends_at,
					promo.created_at,
					promo.updated_at,

					p.id::text,
					p.name,
					p.slug,
					v.id::text,
					v.sku,
					v.price_amount,
					v.compare_at_price_amount,
					v.currency,
					COALESCE(image.url, '')

				FROM promotions promo

				JOIN product_variants v
					ON EXISTS (
						SELECT 1
						FROM promotion_targets target
						WHERE
							target.promotion_id = promo.id
							AND (
								target.variant_id = v.id
								OR target.product_id = v.product_id
							)
					)

				JOIN products p
					ON p.id = v.product_id

				LEFT JOIN LATERAL (
					SELECT pi.url
					FROM product_images pi
					WHERE
						pi.product_id = p.id
						AND pi.is_primary = true
					ORDER BY pi.created_at, pi.id
					LIMIT 1
				) image
					ON true

				WHERE
					promo.scope = 'product'
					AND promo.campaign_type = 'flash_sale'
					AND promo.code IS NULL
					AND promo.status = 'active'
					AND promo.currency = $1
					AND promo.starts_at <= $2
					AND promo.ends_at > $2
					AND p.status = 'active'
					AND v.is_active = true
					AND v.currency = $1

				ORDER BY
					promo.ends_at ASC,
					promo.created_at DESC,
					promo.id,
					p.name,
					v.price_amount,
					v.id
			`,
			currency,
			now,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list active flash sale targets: %w",
				err,
			)
	}
	defer rows.Close()

	result := make([]flashSaleTargetRecord, 0)

	for rows.Next() {
		var record flashSaleTargetRecord
		var percentageBPS pgtype.Int4
		var fixedAmount pgtype.Int8
		var maximumDiscountAmount pgtype.Int8
		var startsAt pgtype.Timestamptz
		var endsAt pgtype.Timestamptz
		var compareAtPrice pgtype.Int8

		if err := rows.Scan(
			&record.Promotion.ID,
			&record.Promotion.Name,
			&record.Promotion.Code,
			&record.Promotion.Scope,
			&record.Promotion.CampaignType,
			&record.Promotion.DiscountType,
			&percentageBPS,
			&fixedAmount,
			&record.Promotion.MinimumSubtotalAmount,
			&maximumDiscountAmount,
			&record.Promotion.Currency,
			&record.Promotion.Status,
			&startsAt,
			&endsAt,
			&record.Promotion.CreatedAt,
			&record.Promotion.UpdatedAt,
			&record.ProductID,
			&record.ProductName,
			&record.ProductSlug,
			&record.VariantID,
			&record.SKU,
			&record.PriceAmount,
			&compareAtPrice,
			&record.Currency,
			&record.PrimaryImageURL,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan active flash sale target: %w",
					err,
				)
		}

		if percentageBPS.Valid {
			value := int(percentageBPS.Int32)
			record.Promotion.PercentageBPS = &value
		}

		if fixedAmount.Valid {
			value := fixedAmount.Int64
			record.Promotion.FixedAmount = &value
		}

		if maximumDiscountAmount.Valid {
			value := maximumDiscountAmount.Int64
			record.Promotion.MaximumDiscountAmount = &value
		}

		if startsAt.Valid {
			value := startsAt.Time
			record.Promotion.StartsAt = &value
		}

		if endsAt.Valid {
			value := endsAt.Time
			record.Promotion.EndsAt = &value
		}

		if compareAtPrice.Valid {
			value := compareAtPrice.Int64
			record.CompareAtPriceAmount = &value
		}

		result = append(result, record)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate active flash sale targets: %w",
				err,
			)
	}

	return result, nil
}
