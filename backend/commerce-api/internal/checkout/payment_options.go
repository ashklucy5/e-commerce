package checkout

import "strings"

const (
	PaymentMethodCOD          = "cod"
	PaymentMethodBKash        = "bkash"
	PaymentMethodNagad        = "nagad"
	PaymentMethodRocket       = "rocket"
	PaymentMethodBankTransfer = "bank_transfer"
)

type PaymentMethodConfig struct {
	CODEnabled bool

	BKashEnabled bool

	NagadEnabled bool

	RocketEnabled bool

	BankTransferEnabled bool

	// ProviderAvailable is required for provider-backed
	// payment methods.
	//
	// The environment flag alone is not sufficient.
	//
	// COD deliberately does not use this callback because
	// it does not require an external payment provider.
	ProviderAvailable func(
		method string,
	) bool
}

type PaymentOption struct {
	Code string `json:"code"`

	Label string `json:"label"`

	RequiresImmediatePayment bool `json:"requires_immediate_payment"`

	PayableAmount int64 `json:"payable_amount"`

	Currency string `json:"currency"`
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

func PaymentOptions(
	currency string,
	totalAmount int64,
) []PaymentOption {
	return []PaymentOption{
		{
			Code: PaymentMethodCOD,

			Label: "Cash on Delivery",

			RequiresImmediatePayment: false,

			PayableAmount: totalAmount,

			Currency: currency,
		},
		{
			Code: PaymentMethodBKash,

			Label: "bKash",

			RequiresImmediatePayment: true,

			PayableAmount: totalAmount,

			Currency: currency,
		},
		{
			Code: PaymentMethodNagad,

			Label: "Nagad",

			RequiresImmediatePayment: true,

			PayableAmount: totalAmount,

			Currency: currency,
		},
		{
			Code: PaymentMethodRocket,

			Label: "Rocket",

			RequiresImmediatePayment: true,

			PayableAmount: totalAmount,

			Currency: currency,
		},
		{
			Code: PaymentMethodBankTransfer,

			Label: "Bank Transfer",

			RequiresImmediatePayment: false,

			PayableAmount: totalAmount,

			Currency: currency,
		},
	}
}

func EnabledPaymentOptions(
	config PaymentMethodConfig,
	currency string,
	totalAmount int64,
) []PaymentOption {
	all :=
		PaymentOptions(
			currency,
			totalAmount,
		)

	enabled :=
		make(
			[]PaymentOption,
			0,
			len(all),
		)

	for _, option := range all {

		if !config.IsEnabled(
			option.Code,
		) {
			continue
		}

		enabled =
			append(
				enabled,
				option,
			)
	}

	return enabled
}

func normalizePaymentMethod(
	value string,
) (string, error) {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if value == "" {
		return "", nil
	}

	switch value {
	case PaymentMethodCOD,
		PaymentMethodBKash,
		PaymentMethodNagad,
		PaymentMethodRocket,
		PaymentMethodBankTransfer:

		return value, nil

	default:
		return "",
			ErrInvalidPaymentMethod
	}
}
