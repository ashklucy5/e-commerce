package promotion

import (
	"strings"
	"time"
)

func IsEligible(
	promotion Promotion,
	subtotalAmount int64,
	currency string,
	now time.Time,
) bool {
	if promotion.Status !=
		StatusActive {
		return false
	}

	if subtotalAmount <
		promotion.MinimumSubtotalAmount {
		return false
	}

	if strings.ToUpper(
		strings.TrimSpace(
			currency,
		),
	) != promotion.Currency {
		return false
	}

	if promotion.StartsAt != nil &&
		now.Before(
			*promotion.StartsAt,
		) {
		return false
	}

	if promotion.EndsAt != nil &&
		!now.Before(
			*promotion.EndsAt,
		) {
		return false
	}

	return true
}
