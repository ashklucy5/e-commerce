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
	platformvalidation "project.local/commerce-api/internal/platform/validation"
	promotiondomain "project.local/commerce-api/internal/promotion"
)

const (
	adminEventPromotionCreated = "promotion_created"
	adminEventPromotionUpdated = "promotion_updated"

	adminPromotionModeAutomatic = "automatic"
	adminPromotionModeCode      = "code"

	adminPromotionStateDraft     = "draft"
	adminPromotionStateDisabled  = "disabled"
	adminPromotionStateScheduled = "scheduled"
	adminPromotionStateLive      = "live"
	adminPromotionStateExpired   = "expired"
)

var (
	ErrAdminPromotionNotFound = errors.New(
		"Admin promotion not found",
	)

	ErrAdminPromotionConflict = errors.New(
		"Admin promotion conflicts with an existing promotion",
	)

	ErrInvalidAdminPromotion = errors.New(
		"invalid Admin promotion configuration",
	)
)

type PromotionReadFilter struct {
	Status       string
	Mode         string
	Scope        string
	CampaignType string

	Query   string
	QueryID string
}

type AdminPromotion struct {
	ID string `json:"id"`

	Name string `json:"name"`
	Code string `json:"code,omitempty"`
	Mode string `json:"mode"`

	Scope        string `json:"scope"`
	CampaignType string `json:"campaign_type"`

	DiscountType string `json:"discount_type"`

	PercentageBPS *int   `json:"percentage_bps,omitempty"`
	FixedAmount   *int64 `json:"fixed_amount,omitempty"`

	MinimumSubtotalAmount int64  `json:"minimum_subtotal_amount"`
	MaximumDiscountAmount *int64 `json:"maximum_discount_amount,omitempty"`

	Currency string `json:"currency"`
	Status   string `json:"status"`

	EffectiveState string `json:"effective_state"`

	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`

	TargetCount int                    `json:"target_count"`
	Targets     []AdminPromotionTarget `json:"targets,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminPromotionTarget struct {
	Type string `json:"type"`

	ProductID   string `json:"product_id,omitempty"`
	ProductName string `json:"product_name,omitempty"`

	VariantID string `json:"variant_id,omitempty"`
	SKU       string `json:"sku,omitempty"`
}

type PromotionTargetInput struct {
	ProductID string
	VariantID string
}

type PromotionListResult struct {
	Items []AdminPromotion
	Meta  platformpagination.Meta
}

type PromotionConfigInput struct {
	Name string

	Code *string

	Scope        string
	CampaignType string

	DiscountType string

	PercentageBPS *int
	FixedAmount   *int64

	MinimumSubtotalAmount int64
	MaximumDiscountAmount *int64

	Currency string
	Status   string

	StartsAt *time.Time
	EndsAt   *time.Time

	Targets []PromotionTargetInput
}

func (s *Service) ListPromotions(
	ctx context.Context,
	params platformpagination.Params,
	filter PromotionReadFilter,
) (PromotionListResult, error) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	rows, err := s.db.Query(
		queryCtx,
		`
			SELECT
				p.id::text,
				p.name,
				COALESCE(p.code, ''),
				p.scope,
				p.campaign_type,
				p.discount_type,
				p.percentage_bps,
				p.fixed_amount,
				p.minimum_subtotal_amount,
				p.maximum_discount_amount,
				p.currency,
				p.status,
				p.starts_at,
				p.ends_at,
				p.created_at,
				p.updated_at,
				(
					SELECT COUNT(*)::int
					FROM promotion_targets target
					WHERE target.promotion_id = p.id
				),
				COUNT(*) OVER()::bigint
			FROM promotions p
			WHERE
				($1 = '' OR p.status = $1)
				AND (
					$2 = ''
					OR ($2 = 'automatic' AND p.code IS NULL)
					OR ($2 = 'code' AND p.code IS NOT NULL)
				)
				AND ($3 = '' OR p.scope = $3)
				AND ($4 = '' OR p.campaign_type = $4)
				AND (
					$5 = ''
					OR p.id = NULLIF($6, '')::uuid
					OR p.code = upper($5)
				)
			ORDER BY p.updated_at DESC, p.id DESC
			LIMIT $7
			OFFSET $8
		`,
		filter.Status,
		filter.Mode,
		filter.Scope,
		filter.CampaignType,
		filter.Query,
		filter.QueryID,
		params.Limit,
		params.Offset(),
	)
	if err != nil {
		return PromotionListResult{},
			fmt.Errorf("list Admin promotions: %w", err)
	}
	defer rows.Close()

	items := make([]AdminPromotion, 0, params.Limit)
	var total int64
	now := time.Now().UTC()

	for rows.Next() {
		var item AdminPromotion
		if err := scanAdminPromotion(rows, &item, &total); err != nil {
			return PromotionListResult{},
				fmt.Errorf("scan Admin promotion: %w", err)
		}

		finalizeAdminPromotion(&item, now)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return PromotionListResult{},
			fmt.Errorf("iterate Admin promotions: %w", err)
	}

	return PromotionListResult{
		Items: items,
		Meta:  platformpagination.NewMeta(params, total),
	}, nil
}

func (s *Service) GetPromotion(
	ctx context.Context,
	promotionID string,
) (AdminPromotion, error) {
	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	return getAdminPromotion(
		queryCtx,
		s.db,
		promotionID,
	)
}

func (s *Service) CreatePromotion(
	ctx context.Context,
	input PromotionConfigInput,
	metadata AdminActionMetadata,
) (AdminPromotion, error) {
	normalized, err := normalizeAdminPromotionInput(input)
	if err != nil {
		return AdminPromotion{}, err
	}

	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	var promotionID string

	err = platformdatabase.WithinTxOptions(
		queryCtx,
		s.db,
		pgx.TxOptions{},
		func(ctx context.Context, tx pgx.Tx) error {
			err := tx.QueryRow(
				ctx,
				`
					INSERT INTO promotions (
						name,
						code,
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
					)
					VALUES (
						$1,
						NULLIF($2, ''),
						$3,
						$4,
						$5,
						$6,
						$7,
						$8,
						$9,
						$10,
						$11,
						$12,
						$13,
						now(),
						now()
					)
					RETURNING id::text
				`,
				normalized.Name,
				adminPromotionCodeValue(normalized.Code),
				normalized.Scope,
				normalized.CampaignType,
				normalized.DiscountType,
				normalized.PercentageBPS,
				normalized.FixedAmount,
				normalized.MinimumSubtotalAmount,
				normalized.MaximumDiscountAmount,
				normalized.Currency,
				normalized.Status,
				normalized.StartsAt,
				normalized.EndsAt,
			).Scan(&promotionID)
			if err != nil {
				if isAdminUniqueViolation(err) {
					return ErrAdminPromotionConflict
				}

				return fmt.Errorf("create Admin promotion: %w", err)
			}

			if err := replaceAdminPromotionTargetsTx(
				ctx,
				tx,
				promotionID,
				normalized.Scope,
				normalized.Targets,
			); err != nil {
				return err
			}

			if err := validateAdminFlashSaleTargetOverlapTx(
				ctx,
				tx,
				promotionID,
				normalized,
			); err != nil {
				return err
			}

			return insertAdminActionAuditTx(
				ctx,
				tx,
				metadata,
				adminEventPromotionCreated,
				map[string]any{
					"promotion_id":  promotionID,
					"name":          normalized.Name,
					"code":          adminPromotionCodeValue(normalized.Code),
					"scope":         normalized.Scope,
					"campaign_type": normalized.CampaignType,
					"discount_type": normalized.DiscountType,
					"status":        normalized.Status,
					"target_count":  len(normalized.Targets),
				},
			)
		},
	)
	if err != nil {
		return AdminPromotion{}, err
	}

	return s.GetPromotion(ctx, promotionID)
}

func (s *Service) ReplacePromotion(
	ctx context.Context,
	promotionID string,
	input PromotionConfigInput,
	metadata AdminActionMetadata,
) (AdminPromotion, error) {
	normalized, err := normalizeAdminPromotionInput(input)
	if err != nil {
		return AdminPromotion{}, err
	}

	queryCtx, cancel := adminReadContext(ctx)
	defer cancel()

	err = platformdatabase.WithinTxOptions(
		queryCtx,
		s.db,
		pgx.TxOptions{},
		func(ctx context.Context, tx pgx.Tx) error {
			var previousName string
			var previousCode string
			var previousStatus string
			var previousDiscountType string
			var previousScope string
			var previousCampaignType string

			err := tx.QueryRow(
				ctx,
				`
					SELECT
						name,
						COALESCE(code, ''),
						status,
						discount_type,
						scope,
						campaign_type
					FROM promotions
					WHERE id = $1::uuid
					FOR UPDATE
				`,
				promotionID,
			).Scan(
				&previousName,
				&previousCode,
				&previousStatus,
				&previousDiscountType,
				&previousScope,
				&previousCampaignType,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAdminPromotionNotFound
			}
			if err != nil {
				return fmt.Errorf("lock Admin promotion: %w", err)
			}

			_, err = tx.Exec(
				ctx,
				`
					UPDATE promotions
					SET
						name = $2,
						code = NULLIF($3, ''),
						scope = $4,
						campaign_type = $5,
						discount_type = $6,
						percentage_bps = $7,
						fixed_amount = $8,
						minimum_subtotal_amount = $9,
						maximum_discount_amount = $10,
						currency = $11,
						status = $12,
						starts_at = $13,
						ends_at = $14,
						updated_at = now()
					WHERE id = $1::uuid
				`,
				promotionID,
				normalized.Name,
				adminPromotionCodeValue(normalized.Code),
				normalized.Scope,
				normalized.CampaignType,
				normalized.DiscountType,
				normalized.PercentageBPS,
				normalized.FixedAmount,
				normalized.MinimumSubtotalAmount,
				normalized.MaximumDiscountAmount,
				normalized.Currency,
				normalized.Status,
				normalized.StartsAt,
				normalized.EndsAt,
			)
			if err != nil {
				if isAdminUniqueViolation(err) {
					return ErrAdminPromotionConflict
				}

				return fmt.Errorf("update Admin promotion: %w", err)
			}

			if err := replaceAdminPromotionTargetsTx(
				ctx,
				tx,
				promotionID,
				normalized.Scope,
				normalized.Targets,
			); err != nil {
				return err
			}

			if err := validateAdminFlashSaleTargetOverlapTx(
				ctx,
				tx,
				promotionID,
				normalized,
			); err != nil {
				return err
			}

			return insertAdminActionAuditTx(
				ctx,
				tx,
				metadata,
				adminEventPromotionUpdated,
				map[string]any{
					"promotion_id":           promotionID,
					"previous_name":          previousName,
					"new_name":               normalized.Name,
					"previous_code":          previousCode,
					"new_code":               adminPromotionCodeValue(normalized.Code),
					"previous_status":        previousStatus,
					"new_status":             normalized.Status,
					"previous_discount_type": previousDiscountType,
					"new_discount_type":      normalized.DiscountType,
					"previous_scope":         previousScope,
					"new_scope":              normalized.Scope,
					"previous_campaign_type": previousCampaignType,
					"new_campaign_type":      normalized.CampaignType,
					"target_count":           len(normalized.Targets),
				},
			)
		},
	)
	if err != nil {
		return AdminPromotion{}, err
	}

	return s.GetPromotion(ctx, promotionID)
}

func getAdminPromotion(
	ctx context.Context,
	querier adminReadQuerier,
	promotionID string,
) (AdminPromotion, error) {
	var result AdminPromotion

	err := scanAdminPromotion(
		querier.QueryRow(
			ctx,
			`
				SELECT
					p.id::text,
					p.name,
					COALESCE(p.code, ''),
					p.scope,
					p.campaign_type,
					p.discount_type,
					p.percentage_bps,
					p.fixed_amount,
					p.minimum_subtotal_amount,
					p.maximum_discount_amount,
					p.currency,
					p.status,
					p.starts_at,
					p.ends_at,
					p.created_at,
					p.updated_at,
					(
						SELECT COUNT(*)::int
						FROM promotion_targets target
						WHERE target.promotion_id = p.id
					)
				FROM promotions p
				WHERE p.id = $1::uuid
			`,
			promotionID,
		),
		&result,
		nil,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminPromotion{}, ErrAdminPromotionNotFound
	}
	if err != nil {
		return AdminPromotion{},
			fmt.Errorf("get Admin promotion: %w", err)
	}

	targets, err := listAdminPromotionTargets(
		ctx,
		querier,
		promotionID,
	)
	if err != nil {
		return AdminPromotion{}, err
	}
	result.Targets = targets

	finalizeAdminPromotion(&result, time.Now().UTC())
	return result, nil
}

type adminPromotionScanner interface {
	Scan(dest ...any) error
}

func scanAdminPromotion(
	row adminPromotionScanner,
	result *AdminPromotion,
	total *int64,
) error {
	var percentageBPS pgtype.Int4
	var fixedAmount pgtype.Int8
	var maximumDiscountAmount pgtype.Int8
	var startsAt pgtype.Timestamptz
	var endsAt pgtype.Timestamptz

	destinations := []any{
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
		&result.TargetCount,
	}

	if total != nil {
		destinations = append(destinations, total)
	}

	if err := row.Scan(destinations...); err != nil {
		return err
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

	result.StartsAt = adminTimePointer(startsAt)
	result.EndsAt = adminTimePointer(endsAt)

	return nil
}

func listAdminPromotionTargets(
	ctx context.Context,
	querier adminReadQuerier,
	promotionID string,
) ([]AdminPromotionTarget, error) {
	rows, err := querier.Query(
		ctx,
		`
			SELECT
				CASE
					WHEN target.variant_id IS NOT NULL THEN 'variant'
					ELSE 'product'
				END,
				COALESCE(p.id::text, vp.id::text, ''),
				COALESCE(p.name, vp.name, ''),
				COALESCE(v.id::text, ''),
				COALESCE(v.sku, '')
			FROM promotion_targets target
			LEFT JOIN products p
				ON p.id = target.product_id
			LEFT JOIN product_variants v
				ON v.id = target.variant_id
			LEFT JOIN products vp
				ON vp.id = v.product_id
			WHERE target.promotion_id = $1::uuid
			ORDER BY
				CASE WHEN target.product_id IS NOT NULL THEN 0 ELSE 1 END,
				COALESCE(p.name, vp.name, ''),
				COALESCE(v.sku, ''),
				target.id
		`,
		promotionID,
	)
	if err != nil {
		return nil,
			fmt.Errorf("list Admin promotion targets: %w", err)
	}
	defer rows.Close()

	result := make([]AdminPromotionTarget, 0)
	for rows.Next() {
		var item AdminPromotionTarget
		if err := rows.Scan(
			&item.Type,
			&item.ProductID,
			&item.ProductName,
			&item.VariantID,
			&item.SKU,
		); err != nil {
			return nil,
				fmt.Errorf("scan Admin promotion target: %w", err)
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf("iterate Admin promotion targets: %w", err)
	}

	return result, nil
}

func replaceAdminPromotionTargetsTx(
	ctx context.Context,
	tx pgx.Tx,
	promotionID string,
	scope string,
	targets []PromotionTargetInput,
) error {
	if _, err := tx.Exec(
		ctx,
		`DELETE FROM promotion_targets WHERE promotion_id = $1::uuid`,
		promotionID,
	); err != nil {
		return fmt.Errorf("clear Admin promotion targets: %w", err)
	}

	if scope != promotiondomain.ScopeProduct {
		return nil
	}

	for _, target := range targets {
		if target.ProductID != "" {
			tag, err := tx.Exec(
				ctx,
				`
					INSERT INTO promotion_targets (
						promotion_id,
						product_id,
						created_at
					)
					SELECT
						$1::uuid,
						p.id,
						now()
					FROM products p
					WHERE p.id = $2::uuid
				`,
				promotionID,
				target.ProductID,
			)
			if err != nil {
				return fmt.Errorf("insert Admin promotion product target: %w", err)
			}
			if tag.RowsAffected() != 1 {
				return fmt.Errorf(
					"%w: product target %s does not exist",
					ErrInvalidAdminPromotion,
					target.ProductID,
				)
			}
			continue
		}

		tag, err := tx.Exec(
			ctx,
			`
				INSERT INTO promotion_targets (
					promotion_id,
					variant_id,
					created_at
				)
				SELECT
					$1::uuid,
					v.id,
					now()
				FROM product_variants v
				WHERE v.id = $2::uuid
			`,
			promotionID,
			target.VariantID,
		)
		if err != nil {
			return fmt.Errorf("insert Admin promotion variant target: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf(
				"%w: variant target %s does not exist",
				ErrInvalidAdminPromotion,
				target.VariantID,
			)
		}
	}

	return nil
}

func normalizeAdminPromotionInput(
	input PromotionConfigInput,
) (PromotionConfigInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Scope = strings.ToLower(strings.TrimSpace(input.Scope))
	input.CampaignType = strings.ToLower(strings.TrimSpace(input.CampaignType))
	input.DiscountType = strings.ToLower(strings.TrimSpace(input.DiscountType))
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))

	if input.Scope == "" {
		input.Scope = promotiondomain.ScopeOrder
	}
	if input.CampaignType == "" {
		input.CampaignType = promotiondomain.CampaignTypeStandard
	}

	if input.Code != nil {
		value := strings.ToUpper(strings.TrimSpace(*input.Code))
		if value == "" {
			input.Code = nil
		} else {
			input.Code = &value
		}
	}

	if input.StartsAt != nil {
		value := input.StartsAt.UTC()
		input.StartsAt = &value
	}
	if input.EndsAt != nil {
		value := input.EndsAt.UTC()
		input.EndsAt = &value
	}

	if input.Name == "" || len([]rune(input.Name)) > 160 {
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: name is required and cannot exceed 160 characters",
				ErrInvalidAdminPromotion,
			)
	}

	if input.Code != nil && !validAdminPromotionCode(*input.Code) {
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: code may contain only A-Z, 0-9, dot, hyphen, and underscore and cannot exceed 64 characters",
				ErrInvalidAdminPromotion,
			)
	}

	switch input.Scope {
	case promotiondomain.ScopeOrder,
		promotiondomain.ScopeProduct:
	default:
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: scope must be order or product",
				ErrInvalidAdminPromotion,
			)
	}

	switch input.CampaignType {
	case promotiondomain.CampaignTypeStandard,
		promotiondomain.CampaignTypeFlashSale:
	default:
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: campaign_type must be standard or flash_sale",
				ErrInvalidAdminPromotion,
			)
	}

	switch input.DiscountType {
	case promotiondomain.DiscountTypePercentage:
		if input.PercentageBPS == nil ||
			*input.PercentageBPS < 1 ||
			*input.PercentageBPS > 10000 ||
			input.FixedAmount != nil {
			return PromotionConfigInput{},
				fmt.Errorf(
					"%w: percentage promotions require percentage_bps between 1 and 10000 and fixed_amount must be null",
					ErrInvalidAdminPromotion,
				)
		}

	case promotiondomain.DiscountTypeFixed:
		if input.FixedAmount == nil ||
			*input.FixedAmount <= 0 ||
			input.PercentageBPS != nil {
			return PromotionConfigInput{},
				fmt.Errorf(
					"%w: fixed promotions require a positive fixed_amount and percentage_bps must be null",
					ErrInvalidAdminPromotion,
				)
		}

	default:
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: discount_type must be percentage or fixed",
				ErrInvalidAdminPromotion,
			)
	}

	if input.MinimumSubtotalAmount < 0 {
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: minimum_subtotal_amount cannot be negative",
				ErrInvalidAdminPromotion,
			)
	}

	if input.MaximumDiscountAmount != nil &&
		*input.MaximumDiscountAmount <= 0 {
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: maximum_discount_amount must be positive when supplied",
				ErrInvalidAdminPromotion,
			)
	}

	if !validAdminPromotionCurrency(input.Currency) {
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: currency must be a three-letter uppercase code",
				ErrInvalidAdminPromotion,
			)
	}

	switch input.Status {
	case promotiondomain.StatusDraft,
		promotiondomain.StatusActive,
		promotiondomain.StatusDisabled:
	default:
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: status must be draft, active, or disabled",
				ErrInvalidAdminPromotion,
			)
	}

	if input.StartsAt != nil &&
		input.EndsAt != nil &&
		!input.EndsAt.After(*input.StartsAt) {
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: ends_at must be later than starts_at",
				ErrInvalidAdminPromotion,
			)
	}

	targets, err := normalizeAdminPromotionTargets(input.Targets)
	if err != nil {
		return PromotionConfigInput{}, err
	}
	input.Targets = targets

	if input.Scope == promotiondomain.ScopeOrder && len(input.Targets) > 0 {
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: order-scoped promotions cannot have product or variant targets",
				ErrInvalidAdminPromotion,
			)
	}

	if input.Scope == promotiondomain.ScopeProduct &&
		input.Status == promotiondomain.StatusActive &&
		len(input.Targets) == 0 {
		return PromotionConfigInput{},
			fmt.Errorf(
				"%w: active product-scoped promotions require at least one target",
				ErrInvalidAdminPromotion,
			)
	}

	if input.CampaignType == promotiondomain.CampaignTypeFlashSale {
		if input.Scope != promotiondomain.ScopeProduct {
			return PromotionConfigInput{},
				fmt.Errorf(
					"%w: flash_sale campaigns must use product scope",
					ErrInvalidAdminPromotion,
				)
		}
		if input.Code != nil {
			return PromotionConfigInput{},
				fmt.Errorf(
					"%w: flash_sale campaigns are automatic and cannot use a code",
					ErrInvalidAdminPromotion,
				)
		}
		if input.StartsAt == nil || input.EndsAt == nil {
			return PromotionConfigInput{},
				fmt.Errorf(
					"%w: flash_sale campaigns require starts_at and ends_at",
					ErrInvalidAdminPromotion,
				)
		}
		if input.MinimumSubtotalAmount != 0 {
			return PromotionConfigInput{},
				fmt.Errorf(
					"%w: flash_sale campaigns require minimum_subtotal_amount to be 0",
					ErrInvalidAdminPromotion,
				)
		}
		if input.MaximumDiscountAmount != nil {
			return PromotionConfigInput{},
				fmt.Errorf(
					"%w: flash_sale campaigns cannot use maximum_discount_amount because storefront item pricing must be deterministic",
					ErrInvalidAdminPromotion,
				)
		}
	}

	return input, nil
}

func normalizeAdminPromotionTargets(
	targets []PromotionTargetInput,
) ([]PromotionTargetInput, error) {
	result := make([]PromotionTargetInput, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))

	for _, target := range targets {
		target.ProductID = strings.TrimSpace(target.ProductID)
		target.VariantID = strings.TrimSpace(target.VariantID)

		if (target.ProductID == "") == (target.VariantID == "") {
			return nil,
				fmt.Errorf(
					"%w: each target must contain exactly one of product_id or variant_id",
					ErrInvalidAdminPromotion,
				)
		}

		kind := "product"
		id := target.ProductID
		if target.VariantID != "" {
			kind = "variant"
			id = target.VariantID
		}

		if !platformvalidation.IsUUID(id) {
			return nil,
				fmt.Errorf(
					"%w: %s target must be a valid UUID",
					ErrInvalidAdminPromotion,
					kind,
				)
		}

		key := kind + ":" + id
		if _, exists := seen[key]; exists {
			return nil,
				fmt.Errorf(
					"%w: duplicate %s target %s",
					ErrInvalidAdminPromotion,
					kind,
					id,
				)
		}
		seen[key] = struct{}{}
		result = append(result, target)
	}

	return result, nil
}

func validAdminPromotionCode(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}

	for _, character := range value {
		if character >= 'A' && character <= 'Z' {
			continue
		}
		if character >= '0' && character <= '9' {
			continue
		}
		switch character {
		case '.', '-', '_':
			continue
		}
		return false
	}

	return true
}

func validAdminPromotionCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}

	for index := 0; index < len(value); index++ {
		if value[index] < 'A' || value[index] > 'Z' {
			return false
		}
	}

	return true
}

func adminPromotionCodeValue(code *string) string {
	if code == nil {
		return ""
	}
	return *code
}

func finalizeAdminPromotion(
	item *AdminPromotion,
	now time.Time,
) {
	if item.Code == "" {
		item.Mode = adminPromotionModeAutomatic
	} else {
		item.Mode = adminPromotionModeCode
	}

	switch item.Status {
	case promotiondomain.StatusDraft:
		item.EffectiveState = adminPromotionStateDraft
		return
	case promotiondomain.StatusDisabled:
		item.EffectiveState = adminPromotionStateDisabled
		return
	}

	if item.StartsAt != nil && now.Before(*item.StartsAt) {
		item.EffectiveState = adminPromotionStateScheduled
		return
	}

	if item.EndsAt != nil && !now.Before(*item.EndsAt) {
		item.EffectiveState = adminPromotionStateExpired
		return
	}

	item.EffectiveState = adminPromotionStateLive
}
