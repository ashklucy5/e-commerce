package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"project.local/commerce-api/internal/promotion"
)

type promotionEvaluator interface {
	Evaluate(
		ctx context.Context,
		input promotion.EvaluateInput,
	) (promotion.Result, error)
}

func (s *Service) SetPromotionEvaluator(
	evaluator promotionEvaluator,
) {
	s.promotions =
		evaluator
}

func (s *Service) validateCheckoutPromotion(
	ctx context.Context,
	checkout checkoutSnapshot,
	items []checkoutItemSnapshot,
	now time.Time,
) error {
	promotionID :=
		strings.TrimSpace(
			checkout.PromotionID,
		)

	promotionCode :=
		strings.ToUpper(
			strings.TrimSpace(
				checkout.PromotionCode,
			),
		)

	// Promotion snapshot structure must be internally valid.
	if promotionID == "" {
		if promotionCode != "" ||
			checkout.DiscountAmount != 0 {
			return ErrCheckoutChanged
		}
	} else {
		if checkout.DiscountAmount <= 0 {
			return ErrCheckoutChanged
		}
	}

	// Keep unit tests and non-promotion callers backwards-compatible.
	// In the actual API router this evaluator will always be wired.
	if s.promotions == nil {
		if promotionID != "" {
			return fmt.Errorf(
				"revalidate checkout promotion: promotion evaluator unavailable",
			)
		}

		return nil
	}

	result, err :=
		s.promotions.Evaluate(
			ctx,
			promotion.EvaluateInput{
				Code: promotionCode,

				SubtotalAmount: checkout.SubtotalAmount,

				Currency: checkout.Currency,

				Lines: promotionLinesFromOrderItems(
					items,
				),

				Now: now,
			},
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			promotion.ErrPromotionNotFound,
		),
			errors.Is(
				err,
				promotion.ErrPromotionNotApplicable,
			),
			errors.Is(
				err,
				promotion.ErrInvalidCode,
			),
			errors.Is(
				err,
				promotion.ErrInvalidSubtotal,
			),
			errors.Is(
				err,
				promotion.ErrInvalidCurrency,
			),
			errors.Is(
				err,
				promotion.ErrInvalidPromotion,
			):
			return ErrCheckoutChanged

		default:
			return fmt.Errorf(
				"revalidate checkout promotion: %w",
				err,
			)
		}
	}

	// No promotion was stored on the checkout.
	//
	// If an automatic promotion is applicable now, the checkout has
	// become stale and should be refreshed before ordering.
	if promotionID == "" {
		if result.Applied {
			return ErrCheckoutChanged
		}

		return nil
	}

	// Checkout had a promotion. It must still resolve to exactly the
	// same promotion and exactly the same discount.
	if !result.Applied {
		return ErrCheckoutChanged
	}

	if strings.TrimSpace(
		result.PromotionID,
	) != promotionID {
		return ErrCheckoutChanged
	}

	if strings.ToUpper(
		strings.TrimSpace(
			result.PromotionCode,
		),
	) != promotionCode {
		return ErrCheckoutChanged
	}

	if result.DiscountAmount !=
		checkout.DiscountAmount {
		return ErrCheckoutChanged
	}

	return nil
}

func promotionLinesFromOrderItems(
	items []checkoutItemSnapshot,
) []promotion.EvaluateLine {
	result :=
		make(
			[]promotion.EvaluateLine,
			0,
			len(items),
		)

	for _, item := range items {
		result =
			append(
				result,
				promotion.EvaluateLine{
					VariantID: item.VariantID,

					Quantity: item.Quantity,

					UnitPriceAmount: item.UnitPriceAmount,
				},
			)
	}

	return result
}
