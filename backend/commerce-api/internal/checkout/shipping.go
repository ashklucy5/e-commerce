package checkout

import (
	"fmt"
	"strings"
	"time"
)

const (
	DeliveryMethodStandard = "standard"
	DeliveryMethodSameDay  = "same_day"
	DeliveryMethodPriority = "priority"
)

type DeliveryMethodRule struct {
	Enabled        bool
	ShippingAmount int64
	MinETA         time.Duration
	MaxETA         time.Duration
}

type DeliveryMethodConfig struct {
	Currency string

	Standard DeliveryMethodRule
	SameDay  DeliveryMethodRule
	Priority DeliveryMethodRule
}

type DeliveryOption struct {
	Code  string `json:"code"`
	Label string `json:"label"`

	ShippingAmount int64  `json:"shipping_amount"`
	Currency       string `json:"currency"`

	EstimatedMinMinutes int64 `json:"estimated_min_minutes"`
	EstimatedMaxMinutes int64 `json:"estimated_max_minutes"`

	Selected bool `json:"selected"`
}

func DefaultDeliveryMethodConfig() DeliveryMethodConfig {
	return DeliveryMethodConfig{
		Currency: "BDT",

		Standard: DeliveryMethodRule{
			Enabled:        true,
			ShippingAmount: 0,
			MinETA:         48 * time.Hour,
			MaxETA:         96 * time.Hour,
		},

		SameDay: DeliveryMethodRule{
			Enabled:        false,
			ShippingAmount: 0,
			MinETA:         4 * time.Hour,
			MaxETA:         12 * time.Hour,
		},

		Priority: DeliveryMethodRule{
			Enabled:        false,
			ShippingAmount: 0,
			MinETA:         24 * time.Hour,
			MaxETA:         48 * time.Hour,
		},
	}
}

func (c DeliveryMethodConfig) Validate() error {
	currency :=
		strings.ToUpper(
			strings.TrimSpace(
				c.Currency,
			),
		)

	if len(currency) != 3 {
		return fmt.Errorf(
			"delivery currency must be a 3-letter code",
		)
	}

	enabled :=
		0

	rules :=
		map[string]DeliveryMethodRule{
			DeliveryMethodStandard: c.Standard,
			DeliveryMethodSameDay:  c.SameDay,
			DeliveryMethodPriority: c.Priority,
		}

	for code, rule := range rules {

		if rule.ShippingAmount < 0 {
			return fmt.Errorf(
				"delivery method %s has negative shipping amount",
				code,
			)
		}

		if rule.MinETA < 0 ||
			rule.MaxETA < rule.MinETA {

			return fmt.Errorf(
				"delivery method %s has invalid ETA range",
				code,
			)
		}

		if rule.Enabled {
			enabled++
		}
	}

	if enabled == 0 {
		return fmt.Errorf(
			"at least one delivery method must be enabled",
		)
	}

	return nil
}

func (c DeliveryMethodConfig) Options(
	currency string,
	selected string,
) ([]DeliveryOption, error) {
	if err :=
		c.Validate(); err != nil {

		return nil, err
	}

	currency =
		strings.ToUpper(
			strings.TrimSpace(
				currency,
			),
		)

	configuredCurrency :=
		strings.ToUpper(
			strings.TrimSpace(
				c.Currency,
			),
		)

	if currency !=
		configuredCurrency {

		return []DeliveryOption{},
			nil
	}

	selected, err :=
		normalizeDeliveryMethod(
			selected,
		)
	if err != nil {
		selected = ""
	}

	result :=
		make(
			[]DeliveryOption,
			0,
			3,
		)

	appendRule :=
		func(
			code string,
			label string,
			rule DeliveryMethodRule,
		) {
			if !rule.Enabled {
				return
			}

			result =
				append(
					result,
					DeliveryOption{
						Code: code,

						Label: label,

						ShippingAmount: rule.ShippingAmount,

						Currency: configuredCurrency,

						EstimatedMinMinutes: durationMinutesCeil(
							rule.MinETA,
						),

						EstimatedMaxMinutes: durationMinutesCeil(
							rule.MaxETA,
						),

						Selected: selected ==
							code,
					},
				)
		}

	appendRule(
		DeliveryMethodStandard,
		"Standard Delivery",
		c.Standard,
	)

	appendRule(
		DeliveryMethodSameDay,
		"Same-Day Delivery",
		c.SameDay,
	)

	appendRule(
		DeliveryMethodPriority,
		"Priority Delivery",
		c.Priority,
	)

	return result, nil
}

func (c DeliveryMethodConfig) Quote(
	method string,
	currency string,
) (DeliveryOption, error) {
	method, err :=
		normalizeDeliveryMethod(
			method,
		)
	if err != nil ||
		method == "" {

		return DeliveryOption{},
			ErrInvalidDeliveryMethod
	}

	currency =
		strings.ToUpper(
			strings.TrimSpace(
				currency,
			),
		)

	configuredCurrency :=
		strings.ToUpper(
			strings.TrimSpace(
				c.Currency,
			),
		)

	if err :=
		c.Validate(); err != nil {

		return DeliveryOption{},
			err
	}

	if currency !=
		configuredCurrency {

		return DeliveryOption{},
			ErrInvalidDeliveryMethod
	}

	options, err :=
		c.Options(
			currency,
			method,
		)
	if err != nil {
		return DeliveryOption{},
			err
	}

	for _, option := range options {

		if option.Code ==
			method {

			return option, nil
		}
	}

	return DeliveryOption{},
		ErrInvalidDeliveryMethod
}

func normalizeDeliveryMethod(
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

	case DeliveryMethodStandard,
		DeliveryMethodSameDay,
		DeliveryMethodPriority:

		return value, nil

	default:

		return "",
			ErrInvalidDeliveryMethod
	}
}

func durationMinutesCeil(
	value time.Duration,
) int64 {
	if value <= 0 {
		return 0
	}

	return int64(
		(value + time.Minute - 1) /
			time.Minute,
	)
}
