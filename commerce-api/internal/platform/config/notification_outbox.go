package config

import (
	"fmt"
	"strconv"
	"time"
)

type NotificationOutboxConfig struct {
	Interval  time.Duration
	BatchSize int
}

func LoadNotificationOutboxConfig() (
	NotificationOutboxConfig,
	error,
) {
	interval, err :=
		time.ParseDuration(
			getEnv(
				"NOTIFICATION_OUTBOX_INTERVAL",
				"5s",
			),
		)
	if err != nil {
		return NotificationOutboxConfig{},
			fmt.Errorf(
				"invalid NOTIFICATION_OUTBOX_INTERVAL: %w",
				err,
			)
	}

	if interval <= 0 {
		return NotificationOutboxConfig{},
			fmt.Errorf(
				"NOTIFICATION_OUTBOX_INTERVAL must be greater than zero",
			)
	}

	batchSize, err :=
		strconv.Atoi(
			getEnv(
				"NOTIFICATION_OUTBOX_BATCH_SIZE",
				"100",
			),
		)
	if err != nil {
		return NotificationOutboxConfig{},
			fmt.Errorf(
				"invalid NOTIFICATION_OUTBOX_BATCH_SIZE: %w",
				err,
			)
	}

	if batchSize <= 0 ||
		batchSize > 1000 {

		return NotificationOutboxConfig{},
			fmt.Errorf(
				"NOTIFICATION_OUTBOX_BATCH_SIZE must be between 1 and 1000",
			)
	}

	return NotificationOutboxConfig{
		Interval: interval,

		BatchSize: batchSize,
	}, nil
}
