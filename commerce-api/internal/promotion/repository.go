package promotion

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {
	return &Repository{
		db: db,
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

const promotionSelect = `
	SELECT
		id::text,
		name,
		COALESCE(code, ''),
		scope,
		campaign_type,
		discount_type,
		percentage_bps,
		fixed_amount,
		minimum_subtotal_amount,
		maximum_discount_amount,
		currency,
		status,
		starts_at,
		ends_at,
		created_at,
		updated_at
	FROM promotions
`

func scanPromotion(
	row rowScanner,
) (Promotion, error) {
	var result Promotion

	var percentageBPS pgtype.Int4
	var fixedAmount pgtype.Int8
	var maximumDiscountAmount pgtype.Int8
	var startsAt pgtype.Timestamptz
	var endsAt pgtype.Timestamptz

	err :=
		row.Scan(
			&result.ID,
			&result.Name,
			&result.Code,
			&result.Scope,
			&result.CampaignType,
			&result.DiscountType,
			&percentageBPS,
			&fixedAmount,
			&result.MinimumSubtotalAmount,
			&maximumDiscountAmount,
			&result.Currency,
			&result.Status,
			&startsAt,
			&endsAt,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return Promotion{}, err
	}

	if percentageBPS.Valid {
		value := int(percentageBPS.Int32)
		result.PercentageBPS = &value
	}

	if fixedAmount.Valid {
		value := fixedAmount.Int64
		result.FixedAmount = &value
	}

	if maximumDiscountAmount.Valid {
		value := maximumDiscountAmount.Int64
		result.MaximumDiscountAmount = &value
	}

	if startsAt.Valid {
		value := startsAt.Time
		result.StartsAt = &value
	}

	if endsAt.Valid {
		value := endsAt.Time
		result.EndsAt = &value
	}

	return result, nil
}

func (r *Repository) GetByCode(
	ctx context.Context,
	code string,
) (Promotion, error) {
	result, err :=
		scanPromotion(
			r.db.QueryRow(
				ctx,
				promotionSelect+`
					WHERE code = $1
					LIMIT 1
				`,
				code,
			),
		)

	if errors.Is(err, pgx.ErrNoRows) {
		return Promotion{}, ErrPromotionNotFound
	}

	if err != nil {
		return Promotion{},
			fmt.Errorf(
				"get promotion by code: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ListAutomatic(
	ctx context.Context,
) ([]Promotion, error) {
	rows, err :=
		r.db.Query(
			ctx,
			promotionSelect+`
				WHERE
					code IS NULL
					AND status = 'active'
				ORDER BY
					created_at,
					id
			`,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list automatic promotions: %w",
				err,
			)
	}
	defer rows.Close()

	result := make([]Promotion, 0)

	for rows.Next() {
		promotion, err := scanPromotion(rows)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan automatic promotion: %w",
					err,
				)
		}

		result = append(result, promotion)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate automatic promotions: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ResolveEligibleVariantIDs(
	ctx context.Context,
	promotionID string,
	variantIDs []string,
) (map[string]struct{}, error) {
	result := make(map[string]struct{})

	if len(variantIDs) == 0 {
		return result, nil
	}

	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT v.id::text
				FROM product_variants v
				WHERE
					v.id::text = ANY($2::text[])
					AND EXISTS (
						SELECT 1
						FROM promotion_targets t
						WHERE
							t.promotion_id = $1::uuid
							AND (
								t.variant_id = v.id
								OR t.product_id = v.product_id
							)
					)
			`,
			promotionID,
			variantIDs,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"resolve promotion target variants: %w",
				err,
			)
	}
	defer rows.Close()

	for rows.Next() {
		var variantID string

		if err := rows.Scan(&variantID); err != nil {
			return nil,
				fmt.Errorf(
					"scan promotion target variant: %w",
					err,
				)
		}

		result[variantID] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate promotion target variants: %w",
				err,
			)
	}

	return result, nil
}
