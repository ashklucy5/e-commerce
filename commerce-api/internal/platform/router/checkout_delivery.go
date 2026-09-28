package router

import (
	"project.local/commerce-api/internal/checkout"
	"project.local/commerce-api/internal/platform/config"
)

func loadCheckoutDeliveryMethods() checkout.DeliveryMethodConfig {
	configured, err :=
		config.LoadCheckoutDeliveryConfig()
	if err != nil {
		panic(
			"invalid checkout delivery configuration: " +
				err.Error(),
		)
	}

	result :=
		checkout.DeliveryMethodConfig{
			Currency: configured.Currency,

			Standard: checkout.DeliveryMethodRule{
				Enabled: configured.Standard.Enabled,

				ShippingAmount: configured.Standard.ShippingAmount,

				MinETA: configured.Standard.MinETA,

				MaxETA: configured.Standard.MaxETA,
			},

			SameDay: checkout.DeliveryMethodRule{
				Enabled: configured.SameDay.Enabled,

				ShippingAmount: configured.SameDay.ShippingAmount,

				MinETA: configured.SameDay.MinETA,

				MaxETA: configured.SameDay.MaxETA,
			},

			Priority: checkout.DeliveryMethodRule{
				Enabled: configured.Priority.Enabled,

				ShippingAmount: configured.Priority.ShippingAmount,

				MinETA: configured.Priority.MinETA,

				MaxETA: configured.Priority.MaxETA,
			},
		}

	if err :=
		result.Validate(); err != nil {

		panic(
			"invalid checkout delivery policy: " +
				err.Error(),
		)
	}

	return result
}
