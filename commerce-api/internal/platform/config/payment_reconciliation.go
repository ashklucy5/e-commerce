package config

import (
	"fmt"
	"strconv"
	"time"
)

type PaymentReconciliationConfig struct {
	Interval  time.Duration
	BatchSize int
}

func LoadPaymentReconciliationConfig() (
	PaymentReconciliationConfig,
	error,
) {
	interval, err :=
		time.ParseDuration(
			getEnv(
				"PAYMENT_RECONCILIATION_INTERVAL",
				"30s",
			),
		)
	if err != nil {
		return PaymentReconciliationConfig{},
			fmt.Errorf(
				"invalid PAYMENT_RECONCILIATION_INTERVAL: %w",
				err,
			)
	}

	if interval <= 0 {
		return PaymentReconciliationConfig{},
			fmt.Errorf(
				"PAYMENT_RECONCILIATION_INTERVAL must be greater than zero",
			)
	}

	batchSize, err :=
		strconv.Atoi(
			getEnv(
				"PAYMENT_RECONCILIATION_BATCH_SIZE",
				"100",
			),
		)
	if err != nil {
		return PaymentReconciliationConfig{},
			fmt.Errorf(
				"invalid PAYMENT_RECONCILIATION_BATCH_SIZE: %w",
				err,
			)
	}

	if batchSize <= 0 ||
		batchSize > 1000 {

		return PaymentReconciliationConfig{},
			fmt.Errorf(
				"PAYMENT_RECONCILIATION_BATCH_SIZE must be between 1 and 1000",
			)
	}

	return PaymentReconciliationConfig{
		Interval:  interval,
		BatchSize: batchSize,
	}, nil
}
