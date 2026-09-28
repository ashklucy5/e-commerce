package promotion

import (
	"testing"
	"time"
)

func TestIsEligible(
	t *testing.T,
) {
	now :=
		time.Date(
			2026,
			time.August,
			18,
			12,
			0,
			0,
			0,
			time.UTC,
		)

	startsAt :=
		now.Add(
			-time.Hour,
		)

	endsAt :=
		now.Add(
			time.Hour,
		)

	promotion :=
		Promotion{
			Status: StatusActive,

			Currency: "BDT",

			MinimumSubtotalAmount: 100000,

			StartsAt: &startsAt,

			EndsAt: &endsAt,
		}

	if !IsEligible(
		promotion,
		200000,
		"BDT",
		now,
	) {
		t.Fatal(
			"expected promotion to be eligible",
		)
	}
}

func TestIsEligibleRejectsExpiredPromotion(
	t *testing.T,
) {
	now :=
		time.Date(
			2026,
			time.August,
			18,
			12,
			0,
			0,
			0,
			time.UTC,
		)

	endsAt := now

	promotion :=
		Promotion{
			Status: StatusActive,

			Currency: "BDT",

			EndsAt: &endsAt,
		}

	if IsEligible(
		promotion,
		200000,
		"BDT",
		now,
	) {
		t.Fatal(
			"expected expired promotion to be ineligible",
		)
	}
}

func TestIsEligibleRejectsBelowMinimumSubtotal(
	t *testing.T,
) {
	promotion :=
		Promotion{
			Status: StatusActive,

			Currency: "BDT",

			MinimumSubtotalAmount: 500000,
		}

	if IsEligible(
		promotion,
		499999,
		"BDT",
		time.Now().UTC(),
	) {
		t.Fatal(
			"expected promotion to be ineligible below minimum subtotal",
		)
	}
}
