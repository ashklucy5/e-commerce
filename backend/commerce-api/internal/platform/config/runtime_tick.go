package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultRuntimeTickMaxMessages = 16
	DefaultRuntimeTickTimeout     = 20 * time.Second
	DefaultRuntimeTickLockTTL     = 30 * time.Second

	MaxRuntimeTickMessages = 256
	MaxRuntimeTickTimeout  = 25 * time.Second
)

type RuntimeTickConfig struct {
	Enabled bool

	BearerToken string

	MaxMessages int

	Timeout time.Duration

	LockTTL time.Duration
}

func LoadRuntimeTickConfig() (
	RuntimeTickConfig,
	error,
) {
	enabled, err :=
		getBoolEnv(
			"RUNTIME_TICK_ENABLED",
			false,
		)
	if err != nil {
		return RuntimeTickConfig{}, err
	}

	bearerToken :=
		strings.TrimSpace(
			os.Getenv(
				"RUNTIME_TICK_TOKEN",
			),
		)

	/*
		The endpoint is internal infrastructure, not a user-facing API.

		Whenever it is enabled it must always be protected by a
		high-entropy shared secret, including in development.
	*/
	if enabled &&
		len(bearerToken) < 32 {

		return RuntimeTickConfig{},
			fmt.Errorf(
				"RUNTIME_TICK_TOKEN must contain at least 32 characters when runtime tick is enabled",
			)
	}

	maxMessages, err :=
		strconv.Atoi(
			getEnv(
				"RUNTIME_TICK_MAX_MESSAGES",
				strconv.Itoa(
					DefaultRuntimeTickMaxMessages,
				),
			),
		)
	if err != nil {
		return RuntimeTickConfig{},
			fmt.Errorf(
				"invalid RUNTIME_TICK_MAX_MESSAGES: %w",
				err,
			)
	}

	if maxMessages < 1 ||
		maxMessages > MaxRuntimeTickMessages {

		return RuntimeTickConfig{},
			fmt.Errorf(
				"RUNTIME_TICK_MAX_MESSAGES must be between 1 and %d",
				MaxRuntimeTickMessages,
			)
	}

	timeout, err :=
		time.ParseDuration(
			getEnv(
				"RUNTIME_TICK_TIMEOUT",
				DefaultRuntimeTickTimeout.
					String(),
			),
		)
	if err != nil {
		return RuntimeTickConfig{},
			fmt.Errorf(
				"invalid RUNTIME_TICK_TIMEOUT: %w",
				err,
			)
	}

	/*
		The API currently has a 30-second HTTP write timeout.

		Keep the bounded background tick below that limit so the
		invocation has time to serialize and return its response.
	*/
	if timeout <= 0 ||
		timeout > MaxRuntimeTickTimeout {

		return RuntimeTickConfig{},
			fmt.Errorf(
				"RUNTIME_TICK_TIMEOUT must be greater than zero and at most %s",
				MaxRuntimeTickTimeout,
			)
	}

	lockTTL, err :=
		time.ParseDuration(
			getEnv(
				"RUNTIME_TICK_LOCK_TTL",
				DefaultRuntimeTickLockTTL.
					String(),
			),
		)
	if err != nil {
		return RuntimeTickConfig{},
			fmt.Errorf(
				"invalid RUNTIME_TICK_LOCK_TTL: %w",
				err,
			)
	}

	if lockTTL <= timeout {
		return RuntimeTickConfig{},
			fmt.Errorf(
				"RUNTIME_TICK_LOCK_TTL must be greater than RUNTIME_TICK_TIMEOUT",
			)
	}

	return RuntimeTickConfig{
		Enabled: enabled,

		BearerToken: bearerToken,

		MaxMessages: maxMessages,

		Timeout: timeout,

		LockTTL: lockTTL,
	}, nil
}
