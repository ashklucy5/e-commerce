package checkout

import "testing"

func TestOnlinePaymentRequiresProviderAvailability(
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
			"COD should not require provider registry",
		)
	}

	online :=
		[]string{
			PaymentMethodBKash,
			PaymentMethodNagad,
			PaymentMethodRocket,
			PaymentMethodBankTransfer,
		}

	for _, method := range online {

		if config.IsEnabled(
			method,
		) {
			t.Fatalf(
				"%s enabled without provider availability",
				method,
			)
		}
	}
}

func TestOnlinePaymentAvailableWhenFlagAndProviderReady(
	t *testing.T,
) {
	config :=
		PaymentMethodConfig{
			CODEnabled: true,

			BKashEnabled: true,

			NagadEnabled: true,

			RocketEnabled: true,

			BankTransferEnabled: true,

			ProviderAvailable: func(
				method string,
			) bool {
				return method ==
					PaymentMethodBKash
			},
		}

	if !config.IsEnabled(
		PaymentMethodBKash,
	) {
		t.Fatal(
			"ready bKash provider should be available",
		)
	}

	if config.IsEnabled(
		PaymentMethodNagad,
	) {
		t.Fatal(
			"Nagad should remain unavailable",
		)
	}

	if config.IsEnabled(
		PaymentMethodRocket,
	) {
		t.Fatal(
			"Rocket should remain unavailable",
		)
	}

	if config.IsEnabled(
		PaymentMethodBankTransfer,
	) {
		t.Fatal(
			"bank transfer should remain unavailable",
		)
	}
}

func TestProviderCannotOverrideDisabledEnvironmentFlag(
	t *testing.T,
) {
	config :=
		PaymentMethodConfig{
			BKashEnabled: false,

			ProviderAvailable: func(
				string,
			) bool {
				return true
			},
		}

	if config.IsEnabled(
		PaymentMethodBKash,
	) {
		t.Fatal(
			"provider must not override disabled environment flag",
		)
	}
}
