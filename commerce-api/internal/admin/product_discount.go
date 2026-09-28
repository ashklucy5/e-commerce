package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
	promotiondomain "project.local/commerce-api/internal/promotion"
)

const (
	adminEventProductDiscountUpdated = "product_discount_updated"
	adminEventProductDiscountCleared = "product_discount_cleared"
)

var (
	ErrAdminProductDiscountVariantNotFound = errors.New(
		"Admin product discount variant not found",
	)

	ErrInvalidAdminProductDiscount = errors.New(
		"invalid Admin product discount configuration",
	)
)

type ProductDiscountReadFilter struct {
	Query   string
	QueryID string
}

type AdminProductDiscount struct {
	ProductID   string `json:"product_id"`
	ProductCode string `json:"product_code"`
	ProductName string `json:"product_name"`

	VariantID string `json:"variant_id"`
	SKU       string `json:"sku"`

	DiscountActive bool `json:"discount_active"`

	RegularPriceAmount int64 `json:"regular_price_amount"`
	SalePriceAmount    int64 `json:"sale_price_amount"`
	DiscountAmount     int64 `json:"discount_amount"`
	DiscountBPS        int   `json:"discount_bps"`

	Currency string `json:"currency"`

	VariantActive bool   `json:"variant_active"`
	ProductStatus string `json:"product_status"`

	UpdatedAt time.Time `json:"updated_at"`
}

type ProductDiscountListResult struct {
	Items []AdminProductDiscount
	Meta  platformpagination.Meta
}

type ProductDiscountConfigInput struct {
	DiscountType string

	PercentageBPS *int
	FixedAmount   *int64
}

func (s *Service) ListProductDiscounts(
	ctx context.Context,
	params platformpagination.Params,
	filter ProductDiscountReadFilter,
) (ProductDiscountListResult, error) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	rows, err := s.db.Query(
		queryCtx,
		`
			SELECT
				p.id::text,
				p.product_code,
				p.name,
				v.id::text,
				v.sku,
				v.price_amount,
				v.compare_at_price_amount,
				v.currency,
				v.is_active,
				p.status,
				v.updated_at,
				COUNT(*) OVER()::bigint
			FROM product_variants v
			JOIN products p
				ON p.id = v.product_id
			WHERE
				v.compare_at_price_amount IS NOT NULL
				AND v.compare_at_price_amount > v.price_amount
				AND (
					$1 = ''
					OR p.id = NULLIF($2, '')::uuid
					OR v.id = NULLIF($2, '')::uuid
					OR p.product_code ILIKE '%' || $1 || '%'
					OR p.name ILIKE '%' || $1 || '%'
					OR v.sku ILIKE '%' || $1 || '%'
				)
			ORDER BY v.updated_at DESC, v.id DESC
			LIMIT $3
			OFFSET $4
		`,
		filter.Query,
		filter.QueryID,
		params.Limit,
		params.Offset(),
	)
	if err != nil {
		return ProductDiscountListResult{},
			fmt.Errorf("list Admin product discounts: %w", err)
	}
	defer rows.Close()

	items := make([]AdminProductDiscount, 0, params.Limit)
	var total int64

	for rows.Next() {
		item, err := scanAdminProductDiscount(rows, &total)
		if err != nil {
			return ProductDiscountListResult{},
				fmt.Errorf("scan Admin product discount: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return ProductDiscountListResult{},
			fmt.Errorf("iterate Admin product discounts: %w", err)
	}

	return ProductDiscountListResult{
		Items: items,
		Meta:  platformpagination.NewMeta(params, total),
	}, nil
}

func (s *Service) ApplyProductDiscount(
	ctx context.Context,
	variantID string,
	input ProductDiscountConfigInput,
	metadata AdminActionMetadata,
) (AdminProductDiscount, error) {
	normalized, err := normalizeAdminProductDiscountInput(input)
	if err != nil {
		return AdminProductDiscount{}, err
	}

	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	err = platformdatabase.WithinTxOptions(
		queryCtx,
		s.db,
		pgx.TxOptions{},
		func(ctx context.Context, tx pgx.Tx) error {
			var productID string
			var productCode string
			var productName string
			var sku string
			var currentPrice int64
			var compareAt pgtype.Int8
			var currency string

			err := tx.QueryRow(
				ctx,
				`
					SELECT
						p.id::text,
						p.product_code,
						p.name,
						v.sku,
						v.price_amount,
						v.compare_at_price_amount,
						v.currency
					FROM product_variants v
					JOIN products p
						ON p.id = v.product_id
					WHERE v.id = $1::uuid
					FOR UPDATE OF v
				`,
				variantID,
			).Scan(
				&productID,
				&productCode,
				&productName,
				&sku,
				&currentPrice,
				&compareAt,
				&currency,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAdminProductDiscountVariantNotFound
			}
			if err != nil {
				return fmt.Errorf("lock Admin product discount variant: %w", err)
			}

			regularPrice := currentPrice
			if compareAt.Valid && compareAt.Int64 > currentPrice {
				regularPrice = compareAt.Int64
			}

			calculatorPromotion := promotiondomain.Promotion{
				DiscountType:  normalized.DiscountType,
				PercentageBPS: normalized.PercentageBPS,
				FixedAmount:   normalized.FixedAmount,
			}

			discountAmount, err := promotiondomain.CalculateDiscount(
				calculatorPromotion,
				regularPrice,
			)
			if err != nil || discountAmount <= 0 {
				return fmt.Errorf(
					"%w: discount must reduce the regular price",
					ErrInvalidAdminProductDiscount,
				)
			}

			salePrice := regularPrice - discountAmount

			_, err = tx.Exec(
				ctx,
				`
					UPDATE product_variants
					SET
						price_amount = $2,
						compare_at_price_amount = $3,
						updated_at = now()
					WHERE id = $1::uuid
				`,
				variantID,
				salePrice,
				regularPrice,
			)
			if err != nil {
				return fmt.Errorf("update Admin product discount: %w", err)
			}

			return insertAdminActionAuditTx(
				ctx,
				tx,
				metadata,
				adminEventProductDiscountUpdated,
				map[string]any{
					"product_id":            productID,
					"product_code":          productCode,
					"product_name":          productName,
					"variant_id":            variantID,
					"sku":                   sku,
					"currency":              currency,
					"previous_price_amount": currentPrice,
					"regular_price_amount":  regularPrice,
					"sale_price_amount":     salePrice,
					"discount_type":         normalized.DiscountType,
					"percentage_bps":        normalized.PercentageBPS,
					"fixed_amount":          normalized.FixedAmount,
				},
			)
		},
	)
	if err != nil {
		return AdminProductDiscount{}, err
	}

	return s.getProductDiscount(ctx, variantID)
}

func (s *Service) ClearProductDiscount(
	ctx context.Context,
	variantID string,
	metadata AdminActionMetadata,
) (AdminProductDiscount, error) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	err := platformdatabase.WithinTxOptions(
		queryCtx,
		s.db,
		pgx.TxOptions{},
		func(ctx context.Context, tx pgx.Tx) error {
			var currentPrice int64
			var compareAt pgtype.Int8

			err := tx.QueryRow(
				ctx,
				`
					SELECT price_amount, compare_at_price_amount
					FROM product_variants
					WHERE id = $1::uuid
					FOR UPDATE
				`,
				variantID,
			).Scan(&currentPrice, &compareAt)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAdminProductDiscountVariantNotFound
			}
			if err != nil {
				return fmt.Errorf("lock Admin product discount for clearing: %w", err)
			}

			restoredPrice := currentPrice
			if compareAt.Valid && compareAt.Int64 >= currentPrice {
				restoredPrice = compareAt.Int64
			}

			_, err = tx.Exec(
				ctx,
				`
					UPDATE product_variants
					SET
						price_amount = $2,
						compare_at_price_amount = NULL,
						updated_at = now()
					WHERE id = $1::uuid
				`,
				variantID,
				restoredPrice,
			)
			if err != nil {
				return fmt.Errorf("clear Admin product discount: %w", err)
			}

			return insertAdminActionAuditTx(
				ctx,
				tx,
				metadata,
				adminEventProductDiscountCleared,
				map[string]any{
					"variant_id":            variantID,
					"previous_price_amount": currentPrice,
					"restored_price_amount": restoredPrice,
				},
			)
		},
	)
	if err != nil {
		return AdminProductDiscount{}, err
	}

	return s.getProductDiscount(ctx, variantID)
}

func (s *Service) getProductDiscount(
	ctx context.Context,
	variantID string,
) (AdminProductDiscount, error) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	item, err := scanAdminProductDiscount(
		s.db.QueryRow(
			queryCtx,
			`
				SELECT
					p.id::text,
					p.product_code,
					p.name,
					v.id::text,
					v.sku,
					v.price_amount,
					v.compare_at_price_amount,
					v.currency,
					v.is_active,
					p.status,
					v.updated_at
				FROM product_variants v
				JOIN products p
					ON p.id = v.product_id
				WHERE v.id = $1::uuid
			`,
			variantID,
		),
		nil,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminProductDiscount{}, ErrAdminProductDiscountVariantNotFound
	}
	if err != nil {
		return AdminProductDiscount{},
			fmt.Errorf("get Admin product discount: %w", err)
	}

	return item, nil
}

type adminProductDiscountScanner interface {
	Scan(dest ...any) error
}

func scanAdminProductDiscount(
	row adminProductDiscountScanner,
	total *int64,
) (AdminProductDiscount, error) {
	var result AdminProductDiscount
	var priceAmount int64
	var compareAt pgtype.Int8

	destinations := []any{
		&result.ProductID,
		&result.ProductCode,
		&result.ProductName,
		&result.VariantID,
		&result.SKU,
		&priceAmount,
		&compareAt,
		&result.Currency,
		&result.VariantActive,
		&result.ProductStatus,
		&result.UpdatedAt,
	}
	if total != nil {
		destinations = append(destinations, total)
	}

	if err := row.Scan(destinations...); err != nil {
		return AdminProductDiscount{}, err
	}

	result.SalePriceAmount = priceAmount
	result.RegularPriceAmount = priceAmount

	if compareAt.Valid && compareAt.Int64 > priceAmount {
		result.DiscountActive = true
		result.RegularPriceAmount = compareAt.Int64
		result.DiscountAmount = compareAt.Int64 - priceAmount

		if compareAt.Int64 > 0 {
			result.DiscountBPS = int(
				(result.DiscountAmount*10000 + compareAt.Int64/2) /
					compareAt.Int64,
			)
		}
	}

	return result, nil
}

func normalizeAdminProductDiscountInput(
	input ProductDiscountConfigInput,
) (ProductDiscountConfigInput, error) {
	input.DiscountType = strings.ToLower(strings.TrimSpace(input.DiscountType))

	switch input.DiscountType {
	case promotiondomain.DiscountTypePercentage:
		if input.PercentageBPS == nil ||
			*input.PercentageBPS < 1 ||
			*input.PercentageBPS > 10000 ||
			input.FixedAmount != nil {
			return ProductDiscountConfigInput{},
				fmt.Errorf(
					"%w: percentage discounts require percentage_bps between 1 and 10000 and fixed_amount must be null",
					ErrInvalidAdminProductDiscount,
				)
		}

	case promotiondomain.DiscountTypeFixed:
		if input.FixedAmount == nil ||
			*input.FixedAmount <= 0 ||
			input.PercentageBPS != nil {
			return ProductDiscountConfigInput{},
				fmt.Errorf(
					"%w: fixed discounts require a positive fixed_amount and percentage_bps must be null",
					ErrInvalidAdminProductDiscount,
				)
		}

	default:
		return ProductDiscountConfigInput{},
			fmt.Errorf(
				"%w: discount_type must be percentage or fixed",
				ErrInvalidAdminProductDiscount,
			)
	}

	return input, nil
}
