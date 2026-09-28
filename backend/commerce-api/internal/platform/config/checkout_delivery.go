package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CheckoutDeliveryMethodConfig struct {
	Enabled        bool
	ShippingAmount int64
	MinETA         time.Duration
	MaxETA         time.Duration
}

type CheckoutDeliveryConfig struct {
	Currency string

	Standard CheckoutDeliveryMethodConfig
	SameDay  CheckoutDeliveryMethodConfig
	Priority CheckoutDeliveryMethodConfig
}

func LoadCheckoutDeliveryConfig() (CheckoutDeliveryConfig, error) {
	currency := strings.ToUpper(
		strings.TrimSpace(
			getEnv("DELIVERY_CURRENCY", "BDT"),
		),
	)

	if len(currency) != 3 {
		return CheckoutDeliveryConfig{},
			fmt.Errorf(
				"DELIVERY_CURRENCY must be a 3-letter currency code",
			)
	}

	standard, err :=
		loadCheckoutDeliveryMethodConfig(
			"DELIVERY_STANDARD",
			true,
			0,
			48*time.Hour,
			96*time.Hour,
		)
	if err != nil {
		return CheckoutDeliveryConfig{}, err
	}

	sameDay, err :=
		loadCheckoutDeliveryMethodConfig(
			"DELIVERY_SAME_DAY",
			false,
			0,
			4*time.Hour,
			12*time.Hour,
		)
	if err != nil {
		return CheckoutDeliveryConfig{}, err
	}

	priority, err :=
		loadCheckoutDeliveryMethodConfig(
			"DELIVERY_PRIORITY",
			false,
			0,
			24*time.Hour,
			48*time.Hour,
		)
	if err != nil {
		return CheckoutDeliveryConfig{}, err
	}

	if !standard.Enabled &&
		!sameDay.Enabled &&
		!priority.Enabled {

		return CheckoutDeliveryConfig{},
			fmt.Errorf(
				"at least one checkout delivery method must be enabled",
			)
	}

	return CheckoutDeliveryConfig{
		Currency: currency,
		Standard: standard,
		SameDay:  sameDay,
		Priority: priority,
	}, nil
}

func loadCheckoutDeliveryMethodConfig(
	prefix string,
	defaultEnabled bool,
	defaultShippingAmount int64,
	defaultMinETA time.Duration,
	defaultMaxETA time.Duration,
) (CheckoutDeliveryMethodConfig, error) {
	enabled, err :=
		getBoolEnv(
			prefix+"_ENABLED",
			defaultEnabled,
		)
	if err != nil {
		return CheckoutDeliveryMethodConfig{}, err
	}

	shippingAmount, err :=
		parseNonNegativeInt64Env(
			prefix+"_FEE_AMOUNT",
			defaultShippingAmount,
		)
	if err != nil {
		return CheckoutDeliveryMethodConfig{}, err
	}

	minETA, err :=
		parseNonNegativeDurationEnv(
			prefix+"_MIN_ETA",
			defaultMinETA,
		)
	if err != nil {
		return CheckoutDeliveryMethodConfig{}, err
	}

	maxETA, err :=
		parseNonNegativeDurationEnv(
			prefix+"_MAX_ETA",
			defaultMaxETA,
		)
	if err != nil {
		return CheckoutDeliveryMethodConfig{}, err
	}

	if maxETA < minETA {
		return CheckoutDeliveryMethodConfig{},
			fmt.Errorf(
				"%s_MAX_ETA must be greater than or equal to %s_MIN_ETA",
				prefix,
				prefix,
			)
	}

	return CheckoutDeliveryMethodConfig{
		Enabled:        enabled,
		ShippingAmount: shippingAmount,
		MinETA:         minETA,
		MaxETA:         maxETA,
	}, nil
}

func parseNonNegativeInt64Env(
	key string,
	fallback int64,
) (int64, error) {
	raw :=
		strings.TrimSpace(
			getEnv(
				key,
				strconv.FormatInt(
					fallback,
					10,
				),
			),
		)

	value, err :=
		strconv.ParseInt(
			raw,
			10,
			64,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"invalid %s: %w",
				key,
				err,
			)
	}

	if value < 0 {
		return 0,
			fmt.Errorf(
				"%s must not be negative",
				key,
			)
	}

	return value, nil
}

func parseNonNegativeDurationEnv(
	key string,
	fallback time.Duration,
) (time.Duration, error) {
	raw :=
		strings.TrimSpace(
			getEnv(
				key,
				fallback.String(),
			),
		)

	value, err :=
		time.ParseDuration(
			raw,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"invalid %s: %w",
				key,
				err,
			)
	}

	if value < 0 {
		return 0,
			fmt.Errorf(
				"%s must not be negative",
				key,
			)
	}

	return value, nil
}
