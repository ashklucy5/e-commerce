package promotion

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
)

type Store interface {
	GetByCode(
		ctx context.Context,
		code string,
	) (Promotion, error)

	ListAutomatic(
		ctx context.Context,
	) ([]Promotion, error)

	ResolveEligibleVariantIDs(
		ctx context.Context,
		promotionID string,
		variantIDs []string,
	) (map[string]struct{}, error)
}

type Service struct {
	store Store
}

func NewService(
	store Store,
) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) Evaluate(
	ctx context.Context,
	input EvaluateInput,
) (Result, error) {
	if input.SubtotalAmount < 0 {
		return Result{}, ErrInvalidSubtotal
	}

	currency :=
		strings.ToUpper(
			strings.TrimSpace(
				input.Currency,
			),
		)

	if len(currency) != 3 {
		return Result{}, ErrInvalidCurrency
	}

	now := input.Now

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	code :=
		strings.ToUpper(
			strings.TrimSpace(
				input.Code,
			),
		)

	if code == "" {
		return s.evaluateAutomatic(
			ctx,
			input.SubtotalAmount,
			currency,
			input.Lines,
			now,
		)
	}

	if len(code) > 64 {
		return Result{}, ErrInvalidCode
	}

	codeResult, err :=
		s.evaluateCode(
			ctx,
			code,
			input.SubtotalAmount,
			currency,
			input.Lines,
			now,
		)
	if err != nil {
		// An explicit code must still be valid. Do not silently hide an
		// invalid/not-applicable code behind an automatic promotion.
		return Result{}, err
	}

	automaticResult, err :=
		s.evaluateAutomatic(
			ctx,
			input.SubtotalAmount,
			currency,
			input.Lines,
			now,
		)
	if err != nil {
		return Result{}, err
	}

	// Promotions never stack. If a valid explicit code would reduce the
	// customer's discount, keep the larger automatic discount instead. On
	// an exact tie, preserve the explicitly entered code.
	if automaticResult.Applied &&
		automaticResult.DiscountAmount > codeResult.DiscountAmount {
		return automaticResult, nil
	}

	return codeResult, nil
}

func (s *Service) evaluateCode(
	ctx context.Context,
	code string,
	subtotalAmount int64,
	currency string,
	lines []EvaluateLine,
	now time.Time,
) (Result, error) {
	promotion, err :=
		s.store.GetByCode(
			ctx,
			code,
		)
	if err != nil {
		return Result{}, err
	}

	discountAmount, err :=
		s.evaluatePromotion(
			ctx,
			promotion,
			subtotalAmount,
			currency,
			lines,
			now,
		)
	if err != nil {
		return Result{}, err
	}

	if discountAmount <= 0 {
		return Result{}, ErrPromotionNotApplicable
	}

	return resultFromPromotion(
		promotion,
		discountAmount,
	), nil
}

func (s *Service) evaluateAutomatic(
	ctx context.Context,
	subtotalAmount int64,
	currency string,
	lines []EvaluateLine,
	now time.Time,
) (Result, error) {
	promotions, err :=
		s.store.ListAutomatic(
			ctx,
		)
	if err != nil {
		return Result{}, err
	}

	var best Result

	for _, promotion := range promotions {
		discountAmount, err :=
			s.evaluatePromotion(
				ctx,
				promotion,
				subtotalAmount,
				currency,
				lines,
				now,
			)
		if err != nil {
			if errors.Is(
				err,
				ErrPromotionNotApplicable,
			) {
				continue
			}

			return Result{}, err
		}

		if discountAmount <= 0 {
			continue
		}

		if !best.Applied ||
			discountAmount > best.DiscountAmount {
			best =
				resultFromPromotion(
					promotion,
					discountAmount,
				)
		}
	}

	return best, nil
}

func (s *Service) evaluatePromotion(
	ctx context.Context,
	promotion Promotion,
	subtotalAmount int64,
	currency string,
	lines []EvaluateLine,
	now time.Time,
) (int64, error) {
	scope := normalizedScope(promotion.Scope)
	campaignType := normalizedCampaignType(promotion.CampaignType)

	if scope != ScopeOrder &&
		scope != ScopeProduct {
		return 0, ErrInvalidPromotion
	}

	if campaignType != CampaignTypeStandard &&
		campaignType != CampaignTypeFlashSale {
		return 0, ErrInvalidPromotion
	}

	if campaignType == CampaignTypeFlashSale &&
		scope != ScopeProduct {
		return 0, ErrInvalidPromotion
	}

	promotion.Scope = scope
	promotion.CampaignType = campaignType

	if scope == ScopeOrder {
		if !IsEligible(
			promotion,
			subtotalAmount,
			currency,
			now,
		) {
			return 0, ErrPromotionNotApplicable
		}

		return CalculateDiscount(
			promotion,
			subtotalAmount,
		)
	}

	eligibleSubtotal,
		discountAmount,
		matched,
		err :=
		s.calculateProductScopedDiscount(
			ctx,
			promotion,
			lines,
		)
	if err != nil {
		return 0, err
	}

	if !matched {
		return 0, ErrPromotionNotApplicable
	}

	if !IsEligible(
		promotion,
		eligibleSubtotal,
		currency,
		now,
	) {
		return 0, ErrPromotionNotApplicable
	}

	if discountAmount > subtotalAmount {
		discountAmount = subtotalAmount
	}

	return discountAmount, nil
}

func (s *Service) calculateProductScopedDiscount(
	ctx context.Context,
	promotion Promotion,
	lines []EvaluateLine,
) (
	int64,
	int64,
	bool,
	error,
) {
	variantIDs :=
		make(
			[]string,
			0,
			len(lines),
		)

	for _, line := range lines {
		variantID := strings.TrimSpace(line.VariantID)

		if variantID == "" ||
			line.Quantity <= 0 ||
			line.UnitPriceAmount < 0 {
			return 0, 0, false, ErrInvalidPromotion
		}

		variantIDs = append(
			variantIDs,
			variantID,
		)
	}

	eligibleVariants, err :=
		s.store.ResolveEligibleVariantIDs(
			ctx,
			promotion.ID,
			variantIDs,
		)
	if err != nil {
		return 0, 0, false, err
	}

	if len(eligibleVariants) == 0 {
		return 0, 0, false, nil
	}

	var eligibleSubtotal int64
	var discountAmount int64
	matched := false

	for _, line := range lines {
		variantID := strings.TrimSpace(line.VariantID)

		if _, ok := eligibleVariants[variantID]; !ok {
			continue
		}

		matched = true

		quantity := int64(line.Quantity)

		if line.UnitPriceAmount >
			math.MaxInt64/quantity {
			return 0, 0, false, ErrInvalidPromotion
		}

		lineSubtotal :=
			line.UnitPriceAmount * quantity

		if eligibleSubtotal >
			math.MaxInt64-lineSubtotal {
			return 0, 0, false, ErrInvalidPromotion
		}

		eligibleSubtotal += lineSubtotal

		unitDiscount, err :=
			calculateProductUnitDiscount(
				promotion,
				line.UnitPriceAmount,
			)
		if err != nil {
			return 0, 0, false, err
		}

		if unitDiscount >
			math.MaxInt64/quantity {
			return 0, 0, false, ErrInvalidPromotion
		}

		lineDiscount :=
			unitDiscount * quantity

		if discountAmount >
			math.MaxInt64-lineDiscount {
			return 0, 0, false, ErrInvalidPromotion
		}

		discountAmount += lineDiscount
	}

	if promotion.MaximumDiscountAmount != nil {
		if *promotion.MaximumDiscountAmount <= 0 {
			return 0, 0, false, ErrInvalidPromotion
		}

		if discountAmount >
			*promotion.MaximumDiscountAmount {
			discountAmount =
				*promotion.MaximumDiscountAmount
		}
	}

	if discountAmount > eligibleSubtotal {
		discountAmount = eligibleSubtotal
	}

	return eligibleSubtotal,
		discountAmount,
		matched,
		nil
}

func calculateProductUnitDiscount(
	promotion Promotion,
	unitPriceAmount int64,
) (int64, error) {
	if unitPriceAmount < 0 {
		return 0, ErrInvalidSubtotal
	}

	uncapped := promotion
	uncapped.MaximumDiscountAmount = nil

	return CalculateDiscount(
		uncapped,
		unitPriceAmount,
	)
}

func normalizedScope(
	value string,
) string {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" {
		return ScopeOrder
	}

	return value
}

func normalizedCampaignType(
	value string,
) string {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" {
		return CampaignTypeStandard
	}

	return value
}

func resultFromPromotion(
	promotion Promotion,
	discountAmount int64,
) Result {
	return Result{
		Applied: true,

		PromotionID: promotion.ID,

		PromotionCode: promotion.Code,

		PromotionName: promotion.Name,

		Scope: normalizedScope(
			promotion.Scope,
		),

		CampaignType: normalizedCampaignType(
			promotion.CampaignType,
		),

		DiscountType: promotion.DiscountType,

		DiscountAmount: discountAmount,
	}
}
