package promotion

import "testing"

func TestCalculateDiscountPercentage(
	t *testing.T,
) {
	bps := 1000

	promotion :=
		Promotion{
			DiscountType: DiscountTypePercentage,

			PercentageBPS: &bps,
		}

	got, err :=
		CalculateDiscount(
			promotion,
			1323000,
		)
	if err != nil {
		t.Fatalf(
			"CalculateDiscount() error = %v",
			err,
		)
	}

	const want int64 = 132300

	if got != want {
		t.Fatalf(
			"CalculateDiscount() = %d, want %d",
			got,
			want,
		)
	}
}

func TestCalculateDiscountPercentageCap(
	t *testing.T,
) {
	bps := 2500

	capAmount :=
		int64(
			200000,
		)

	promotion :=
		Promotion{
			DiscountType: DiscountTypePercentage,

			PercentageBPS: &bps,

			MaximumDiscountAmount: &capAmount,
		}

	got, err :=
		CalculateDiscount(
			promotion,
			1323000,
		)
	if err != nil {
		t.Fatalf(
			"CalculateDiscount() error = %v",
			err,
		)
	}

	const want int64 = 200000

	if got != want {
		t.Fatalf(
			"CalculateDiscount() = %d, want %d",
			got,
			want,
		)
	}
}

func TestCalculateDiscountFixedCannotExceedSubtotal(
	t *testing.T,
) {
	fixed :=
		int64(
			500000,
		)

	promotion :=
		Promotion{
			DiscountType: DiscountTypeFixed,

			FixedAmount: &fixed,
		}

	got, err :=
		CalculateDiscount(
			promotion,
			200000,
		)
	if err != nil {
		t.Fatalf(
			"CalculateDiscount() error = %v",
			err,
		)
	}

	const want int64 = 200000

	if got != want {
		t.Fatalf(
			"CalculateDiscount() = %d, want %d",
			got,
			want,
		)
	}
}
