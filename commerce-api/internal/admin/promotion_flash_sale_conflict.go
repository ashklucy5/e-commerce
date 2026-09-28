package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	promotiondomain "project.local/commerce-api/internal/promotion"
)

var ErrAdminPromotionTargetConflict = errors.New(
	"Admin promotion targets conflict with an existing flash sale",
)

const adminFlashSaleOverlapAdvisoryLock int64 = 8463027109184321

// validateAdminFlashSaleTargetOverlapTx prevents two active flash-sale
// campaigns in the same currency from targeting the same effective variant
// during overlapping time windows.
//
// Product targets are treated as covering every variant belonging to that
// product. Variant targets cover only that variant. The transaction-scoped
// advisory lock serializes flash-sale overlap validation so two concurrent
// Admin writes cannot both pass the check before either transaction commits.
func validateAdminFlashSaleTargetOverlapTx(
	ctx context.Context,
	tx pgx.Tx,
	promotionID string,
	input PromotionConfigInput,
) error {
	if input.Scope != promotiondomain.ScopeProduct ||
		input.CampaignType != promotiondomain.CampaignTypeFlashSale ||
		input.Status != promotiondomain.StatusActive {
		return nil
	}

	// normalizeAdminPromotionInput validates these requirements before a
	// transaction starts. Keep this guard defensive in case this helper is
	// called from another path later.
	if input.StartsAt == nil ||
		input.EndsAt == nil ||
		len(input.Targets) == 0 {
		return fmt.Errorf(
			"%w: active flash sales require a schedule and at least one target",
			ErrInvalidAdminPromotion,
		)
	}

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock($1)`,
		adminFlashSaleOverlapAdvisoryLock,
	); err != nil {
		return fmt.Errorf(
			"lock Admin flash sale overlap validation: %w",
			err,
		)
	}

	var conflictingPromotionID string
	var conflictingPromotionName string

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				p.id::text,
				p.name
			FROM promotions p
			WHERE
				p.id <> $1::uuid
				AND p.scope = 'product'
				AND p.campaign_type = 'flash_sale'
				AND p.status = 'active'
				AND p.currency = $2
				AND p.starts_at IS NOT NULL
				AND p.ends_at IS NOT NULL
				AND p.starts_at < $4::timestamptz
				AND p.ends_at > $3::timestamptz
				AND EXISTS (
					SELECT 1
					FROM promotion_targets candidate_target
					JOIN promotion_targets existing_target
						ON existing_target.promotion_id = p.id
					LEFT JOIN product_variants candidate_variant
						ON candidate_variant.id = candidate_target.variant_id
					LEFT JOIN product_variants existing_variant
						ON existing_variant.id = existing_target.variant_id
					WHERE
						candidate_target.promotion_id = $1::uuid
						AND (
							(
								candidate_target.product_id IS NOT NULL
								AND existing_target.product_id = candidate_target.product_id
							)
							OR (
								candidate_target.variant_id IS NOT NULL
								AND existing_target.variant_id = candidate_target.variant_id
							)
							OR (
								candidate_target.product_id IS NOT NULL
								AND existing_variant.product_id = candidate_target.product_id
							)
							OR (
								candidate_target.variant_id IS NOT NULL
								AND existing_target.product_id = candidate_variant.product_id
							)
						)
				)
			ORDER BY
				p.starts_at,
				p.id
			LIMIT 1
		`,
		promotionID,
		input.Currency,
		*input.StartsAt,
		*input.EndsAt,
	).Scan(
		&conflictingPromotionID,
		&conflictingPromotionName,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf(
			"check Admin flash sale target overlap: %w",
			err,
		)
	}

	return fmt.Errorf(
		"%w: flash sale overlaps promotion %q (%s) on at least one targeted variant during the same schedule",
		ErrAdminPromotionTargetConflict,
		conflictingPromotionName,
		conflictingPromotionID,
	)
}
