package config

import (
	"fmt"
	"strconv"
	"time"
)

type WorkerConfig struct {
	Concurrency int
	Prefetch    int64

	Block             time.Duration
	HandlerTimeout    time.Duration
	ClaimMinIdle      time.Duration
	ClaimInterval     time.Duration
	RetryPollInterval time.Duration
}

func LoadWorkerConfig() (
	WorkerConfig,
	error,
) {
	concurrency, err :=
		strconv.Atoi(
			getEnv(
				"WORKER_CONCURRENCY",
				"4",
			),
		)
	if err != nil ||
		concurrency <= 0 ||
		concurrency > 64 {

		return WorkerConfig{},
			fmt.Errorf(
				"WORKER_CONCURRENCY must be between 1 and 64",
			)
	}

	prefetch, err :=
		strconv.ParseInt(
			getEnv(
				"WORKER_PREFETCH",
				"16",
			),
			10,
			64,
		)
	if err != nil ||
		prefetch <= 0 ||
		prefetch > 256 {

		return WorkerConfig{},
			fmt.Errorf(
				"WORKER_PREFETCH must be between 1 and 256",
			)
	}

	block, err :=
		time.ParseDuration(
			getEnv(
				"WORKER_QUEUE_BLOCK",
				"2s",
			),
		)
	if err != nil ||
		block <= 0 {

		return WorkerConfig{},
			fmt.Errorf(
				"WORKER_QUEUE_BLOCK must be a positive duration",
			)
	}

	handlerTimeout, err :=
		time.ParseDuration(
			getEnv(
				"WORKER_HANDLER_TIMEOUT",
				"2m",
			),
		)
	if err != nil ||
		handlerTimeout <= 0 {

		return WorkerConfig{},
			fmt.Errorf(
				"WORKER_HANDLER_TIMEOUT must be a positive duration",
			)
	}

	claimMinIdle, err :=
		time.ParseDuration(
			getEnv(
				"WORKER_CLAIM_MIN_IDLE",
				"5m",
			),
		)
	if err != nil ||
		claimMinIdle <= handlerTimeout {

		return WorkerConfig{},
			fmt.Errorf(
				"WORKER_CLAIM_MIN_IDLE must be greater than WORKER_HANDLER_TIMEOUT",
			)
	}

	claimInterval, err :=
		time.ParseDuration(
			getEnv(
				"WORKER_CLAIM_INTERVAL",
				"30s",
			),
		)
	if err != nil ||
		claimInterval <= 0 {

		return WorkerConfig{},
			fmt.Errorf(
				"WORKER_CLAIM_INTERVAL must be a positive duration",
			)
	}

	retryPollInterval, err :=
		time.ParseDuration(
			getEnv(
				"WORKER_RETRY_POLL_INTERVAL",
				"1s",
			),
		)
	if err != nil ||
		retryPollInterval <= 0 {

		return WorkerConfig{},
			fmt.Errorf(
				"WORKER_RETRY_POLL_INTERVAL must be a positive duration",
			)
	}

	return WorkerConfig{
			Concurrency: concurrency,

			Prefetch: prefetch,

			Block: block,

			HandlerTimeout: handlerTimeout,

			ClaimMinIdle: claimMinIdle,

			ClaimInterval: claimInterval,

			RetryPollInterval: retryPollInterval,
		},
		nil
}
