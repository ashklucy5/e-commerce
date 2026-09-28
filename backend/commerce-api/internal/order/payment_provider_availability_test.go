package order

import "testing"

func TestOrderOnlinePaymentRequiresProviderAvailability(
	t *testing.T,
) {
	config :=
		PaymentMethodConfig{
			CODEnabled: true,

			BKashEnabled: true,

			NagadEnabled: true,

			RocketEnabled: true,

			BankTransferEnabled: true,
		}

	if !config.IsEnabled(
		PaymentMethodCOD,
	) {
		t.Fatal(
			"COD should be enabled independently",
		)
	}

	methods :=
		[]string{
			PaymentMethodBKash,
			PaymentMethodNagad,
			PaymentMethodRocket,
			PaymentMethodBankTransfer,
		}

	for _, method := range methods {

		if config.IsEnabled(
			method,
		) {
			t.Fatalf(
				"%s order placement allowed without provider",
				method,
			)
		}
	}
}

func TestOrderOnlinePaymentRequiresBothFlagAndProvider(
	t *testing.T,
) {
	config :=
		PaymentMethodConfig{
			BKashEnabled: true,

			NagadEnabled: false,

			ProviderAvailable: func(
				method string,
			) bool {
				return method ==
					PaymentMethodBKash ||
					method ==
						PaymentMethodNagad
			},
		}

	if !config.IsEnabled(
		PaymentMethodBKash,
	) {
		t.Fatal(
			"bKash should be available",
		)
	}

	if config.IsEnabled(
		PaymentMethodNagad,
	) {
		t.Fatal(
			"disabled Nagad flag must win",
		)
	}
}
