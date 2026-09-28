package router

import (
	"testing"

	paymentintegrations "project.local/commerce-api/integrations/payments"
)

func TestPaymentProviderRegistryRequiresAdapter(
	t *testing.T,
) {
	registry :=
		newPaymentProviderRegistry(
			Dependencies{
				BKashEnabled: true,

				NagadEnabled: true,

				RocketEnabled: true,

				BankTransferEnabled: true,
			},
		)

	codes :=
		[]string{
			paymentintegrations.ProviderBKash,
			paymentintegrations.ProviderNagad,
			paymentintegrations.ProviderRocket,
			paymentintegrations.ProviderBankTransfer,
		}

	for _, code := range codes {

		status :=
			registry.Status(
				code,
			)

		if !status.Enabled {
			t.Fatalf(
				"%s should have environment enablement",
				code,
			)
		}

		if status.Registered {
			t.Fatalf(
				"%s unexpectedly registered",
				code,
			)
		}

		if status.Configured {
			t.Fatalf(
				"%s unexpectedly configured",
				code,
			)
		}

		if status.Available {
			t.Fatalf(
				"%s must not be available without adapter",
				code,
			)
		}

		if registry.Available(
			code,
		) {
			t.Fatalf(
				"%s resolved without adapter",
				code,
			)
		}
	}
}

func TestPaymentProviderRegistryDisabledProvidersRemainUnavailable(
	t *testing.T,
) {
	registry :=
		newPaymentProviderRegistry(
			Dependencies{},
		)

	codes :=
		[]string{
			paymentintegrations.ProviderBKash,
			paymentintegrations.ProviderNagad,
			paymentintegrations.ProviderRocket,
			paymentintegrations.ProviderBankTransfer,
		}

	for _, code := range codes {

		status :=
			registry.Status(
				code,
			)

		if status.Enabled ||
			status.Registered ||
			status.Configured ||
			status.Available {

			t.Fatalf(
				"unexpected status for %s: %#v",
				code,
				status,
			)
		}
	}
}
