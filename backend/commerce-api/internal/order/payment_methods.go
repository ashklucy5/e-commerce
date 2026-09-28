package order

import "strings"

type PaymentMethodConfig struct {
	CODEnabled bool

	BKashEnabled bool

	NagadEnabled bool

	RocketEnabled bool

	BankTransferEnabled bool

	// ProviderAvailable performs the provider-registry
	// availability check for every non-COD method.
	ProviderAvailable func(
		method string,
	) bool
}

func (c PaymentMethodConfig) IsEnabled(
	method string,
) bool {
	method =
		strings.ToLower(
			strings.TrimSpace(
				method,
			),
		)

	switch method {
	case PaymentMethodCOD:
		return c.CODEnabled

	case PaymentMethodBKash:
		return c.providerMethodAvailable(
			PaymentMethodBKash,
			c.BKashEnabled,
		)

	case PaymentMethodNagad:
		return c.providerMethodAvailable(
			PaymentMethodNagad,
			c.NagadEnabled,
		)

	case PaymentMethodRocket:
		return c.providerMethodAvailable(
			PaymentMethodRocket,
			c.RocketEnabled,
		)

	case PaymentMethodBankTransfer:
		return c.providerMethodAvailable(
			PaymentMethodBankTransfer,
			c.BankTransferEnabled,
		)

	default:
		return false
	}
}

func (c PaymentMethodConfig) providerMethodAvailable(
	method string,
	enabled bool,
) bool {
	if !enabled {
		return false
	}

	if c.ProviderAvailable == nil {
		return false
	}

	return c.ProviderAvailable(
		method,
	)
}

// AllPaymentMethodsEnabled is retained primarily for isolated
// service/tests that deliberately want every method available.
//
// Runtime router wiring does not use this configuration.
func AllPaymentMethodsEnabled() PaymentMethodConfig {
	return PaymentMethodConfig{
		CODEnabled: true,

		BKashEnabled: true,

		NagadEnabled: true,

		RocketEnabled: true,

		BankTransferEnabled: true,

		ProviderAvailable: func(
			string,
		) bool {
			return true
		},
	}
}

type PaymentOption struct {
	Code string `json:"code"`

	Label string `json:"label"`

	RequiresImmediatePayment bool `json:"requires_immediate_payment"`

	PayableAmount int64 `json:"payable_amount"`

	Currency string `json:"currency"`
}

func (s *Service) EnabledPaymentOptions(
	currency string,
	totalAmount int64,
) []PaymentOption {
	options := []PaymentOption{
		{
			Code:                     PaymentMethodCOD,
			Label:                    "Cash on Delivery",
			RequiresImmediatePayment: false,
		},
		{
			Code:                     PaymentMethodBKash,
			Label:                    "bKash",
			RequiresImmediatePayment: true,
		},
		{
			Code:                     PaymentMethodNagad,
			Label:                    "Nagad",
			RequiresImmediatePayment: true,
		},
		{
			Code:                     PaymentMethodRocket,
			Label:                    "Rocket",
			RequiresImmediatePayment: true,
		},
		{
			Code:                     PaymentMethodBankTransfer,
			Label:                    "Bank Transfer",
			RequiresImmediatePayment: false,
		},
	}

	result := make(
		[]PaymentOption,
		0,
		len(options),
	)

	for _, option := range options {
		if !s.paymentMethods.IsEnabled(
			option.Code,
		) {
			continue
		}

		option.Currency = currency
		option.PayableAmount = totalAmount

		result = append(
			result,
			option,
		)
	}

	return result
}