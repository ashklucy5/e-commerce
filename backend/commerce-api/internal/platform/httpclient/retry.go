package httpclient

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRetryAttempts = 3

	defaultRetryBaseDelay = 200 * time.Millisecond

	defaultRetryMaxDelay = 2 * time.Second
)

type RetryConfig struct {
	MaxAttempts int

	BaseDelay time.Duration

	MaxDelay time.Duration
}

type retryContextKey struct{}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: defaultRetryAttempts,

		BaseDelay: defaultRetryBaseDelay,

		MaxDelay: defaultRetryMaxDelay,
	}
}

// AllowRetry explicitly permits retrying a request whose HTTP method is
// normally treated as non-idempotent.
//
// Use this only when the remote operation is safe to repeat.
//
// Voyage embedding POST requests are safe because they calculate and
// return embeddings without creating durable remote state.
func AllowRetry(
	request *http.Request,
) *http.Request {
	if request == nil {
		return nil
	}

	ctx :=
		context.WithValue(
			request.Context(),
			retryContextKey{},
			true,
		)

	return request.WithContext(
		ctx,
	)
}

func normalizeRetryConfig(
	cfg RetryConfig,
) RetryConfig {
	defaults :=
		DefaultRetryConfig()

	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts =
			defaults.MaxAttempts
	}

	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay =
			defaults.BaseDelay
	}

	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay =
			defaults.MaxDelay
	}

	if cfg.MaxDelay <
		cfg.BaseDelay {

		cfg.MaxDelay =
			cfg.BaseDelay
	}

	return cfg
}

func requestMayRetry(
	request *http.Request,
) bool {
	if request == nil {
		return false
	}

	switch request.Method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPut,
		http.MethodDelete:

		return true
	}

	value, _ :=
		request.Context().
			Value(
				retryContextKey{},
			).(bool)

	return value
}

func shouldRetryStatus(
	status int,
) bool {
	switch status {
	case http.StatusRequestTimeout,
		http.StatusTooEarly,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:

		return true

	default:
		return false
	}
}

func shouldRetryError(
	err error,
) bool {
	if err == nil {
		return false
	}

	if errors.Is(
		err,
		io.EOF,
	) ||
		errors.Is(
			err,
			io.ErrUnexpectedEOF,
		) {

		return true
	}

	var networkError net.Error

	if errors.As(
		err,
		&networkError,
	) {
		return true
	}

	return false
}

func retryDelay(
	cfg RetryConfig,
	failedAttempt int,
	response *http.Response,
	now time.Time,
) time.Duration {
	if response != nil {
		if value :=
			strings.TrimSpace(
				response.Header.Get(
					"Retry-After",
				),
			); value != "" {

			if seconds, err :=
				strconv.ParseInt(
					value,
					10,
					64,
				); err == nil &&
				seconds >= 0 {

				delay :=
					time.Duration(
						seconds,
					) *
						time.Second

				if delay >
					cfg.MaxDelay {

					return cfg.MaxDelay
				}

				return delay
			}

			if retryAt, err :=
				http.ParseTime(
					value,
				); err == nil {

				delay :=
					retryAt.Sub(
						now,
					)

				if delay < 0 {
					delay = 0
				}

				if delay >
					cfg.MaxDelay {

					return cfg.MaxDelay
				}

				return delay
			}
		}
	}

	delay :=
		cfg.BaseDelay

	for attempt :=
		1; attempt < failedAttempt; attempt++ {

		if delay >=
			cfg.MaxDelay/2 {

			delay =
				cfg.MaxDelay

			break
		}

		delay *= 2
	}

	if delay >
		cfg.MaxDelay {

		delay =
			cfg.MaxDelay
	}

	return delay
}

func waitForRetry(
	ctx context.Context,
	delay time.Duration,
) error {
	if delay <= 0 {
		return nil
	}

	timer :=
		time.NewTimer(
			delay,
		)

	defer timer.Stop()

	select {
	case <-timer.C:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}
