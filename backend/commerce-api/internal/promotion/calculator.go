package promotion

func CalculateDiscount(
	promotion Promotion,
	subtotalAmount int64,
) (int64, error) {
	if subtotalAmount < 0 {
		return 0,
			ErrInvalidSubtotal
	}

	var discount int64

	switch promotion.DiscountType {
	case DiscountTypePercentage:
		if promotion.PercentageBPS == nil ||
			*promotion.PercentageBPS < 1 ||
			*promotion.PercentageBPS > 10000 {
			return 0,
				ErrInvalidPromotion
		}

		bps :=
			int64(
				*promotion.PercentageBPS,
			)

		// Avoid subtotalAmount * bps overflowing int64.
		//
		// Equivalent to:
		//
		// subtotalAmount * bps / 10000
		//
		// but multiplication is performed only on bounded
		// components.
		whole :=
			subtotalAmount /
				10000

		remainder :=
			subtotalAmount %
				10000

		discount =
			whole*bps +
				(remainder*bps)/10000

	case DiscountTypeFixed:
		if promotion.FixedAmount == nil ||
			*promotion.FixedAmount <= 0 {
			return 0,
				ErrInvalidPromotion
		}

		discount =
			*promotion.FixedAmount

	default:
		return 0,
			ErrInvalidPromotion
	}

	if promotion.MaximumDiscountAmount != nil {
		if *promotion.MaximumDiscountAmount <= 0 {
			return 0,
				ErrInvalidPromotion
		}

		if discount >
			*promotion.MaximumDiscountAmount {
			discount =
				*promotion.MaximumDiscountAmount
		}
	}

	if discount >
		subtotalAmount {
		discount =
			subtotalAmount
	}

	if discount < 0 {
		return 0,
			ErrInvalidPromotion
	}

	return discount, nil
}
