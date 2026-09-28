package router

import (
	paymentintegrations "project.local/commerce-api/integrations/payments"
)

func newPaymentProviderRegistry(
	deps Dependencies,
) *paymentintegrations.Registry {
	registry :=
		paymentintegrations.NewRegistry(
			paymentintegrations.Enablement{
				BKash: deps.BKashEnabled,

				Nagad: deps.NagadEnabled,

				Rocket: deps.RocketEnabled,

				BankTransfer: deps.BankTransferEnabled,
			},
		)

	/*
		No external providers are registered yet.

		That is intentional.

		When a real provider integration is added later, this is
		the central place where the configured adapter will be
		registered.

		For example, later:

			if bkashAdapter != nil {
				if err := registry.Register(
					bkashAdapter,
				); err != nil {
					panic(...)
				}
			}

		Until that happens, Registry.Available("bkash")
		remains false even if BKASH_ENABLED=true.
	*/

	return registry
}
