package refund

import "testing"

func TestProviderForPaymentMethod(
	t *testing.T,
) {
	tests := map[string]string{
		"cod":           ProviderManual,
		"bkash":         ProviderBKash,
		"nagad":         ProviderNagad,
		"rocket":        ProviderRocket,
		"bank_transfer": ProviderBankTransfer,
	}

	for paymentMethod, expected := range tests {
		actual, ok :=
			providerForPaymentMethod(
				paymentMethod,
			)

		if !ok {
			t.Fatalf(
				"expected %q to be supported",
				paymentMethod,
			)
		}

		if actual != expected {
			t.Fatalf(
				"payment method %q: expected provider %q, got %q",
				paymentMethod,
				expected,
				actual,
			)
		}
	}
}

func TestCalculateReturnRefundAmount(
	t *testing.T,
) {
	amount, err :=
		calculateReturnRefundAmount(
			378000,
			1323000,
			0,
		)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if amount != 378000 {
		t.Fatalf(
			"expected 378000, got %d",
			amount,
		)
	}
}

func TestCalculateReturnRefundAmountWithDiscount(
	t *testing.T,
) {
	amount, err :=
		calculateReturnRefundAmount(
			50000,
			100000,
			10000,
		)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if amount != 45000 {
		t.Fatalf(
			"expected 45000, got %d",
			amount,
		)
	}
}
