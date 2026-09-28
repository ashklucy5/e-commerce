package httpclient

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New(
	"HTTP circuit breaker is open",
)

const (
	defaultCircuitFailureThreshold = 5

	defaultCircuitOpenTimeout = 30 * time.Second
)

type CircuitBreakerConfig struct {
	FailureThreshold int

	OpenTimeout time.Duration
}

type CircuitBreaker struct {
	mu sync.Mutex

	config CircuitBreakerConfig

	consecutiveFailures int

	openedAt time.Time

	probeInFlight bool
}

func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: defaultCircuitFailureThreshold,

		OpenTimeout: defaultCircuitOpenTimeout,
	}
}

func NewCircuitBreaker(
	cfg CircuitBreakerConfig,
) *CircuitBreaker {
	cfg =
		normalizeCircuitBreakerConfig(
			cfg,
		)

	return &CircuitBreaker{
		config: cfg,
	}
}

func normalizeCircuitBreakerConfig(
	cfg CircuitBreakerConfig,
) CircuitBreakerConfig {
	defaults :=
		DefaultCircuitBreakerConfig()

	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold =
			defaults.FailureThreshold
	}

	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout =
			defaults.OpenTimeout
	}

	return cfg
}

// Before allows normal traffic while the breaker is closed.
//
// After the circuit has been open for OpenTimeout, exactly one caller is
// permitted through as a probe. A successful probe closes the circuit;
// a failed probe opens it again.
func (b *CircuitBreaker) Before(
	now time.Time,
) error {
	if b == nil {
		return nil
	}

	b.mu.Lock()

	defer b.mu.Unlock()

	if b.openedAt.IsZero() {
		return nil
	}

	if now.Sub(
		b.openedAt,
	) <
		b.config.OpenTimeout {

		return ErrCircuitOpen
	}

	if b.probeInFlight {
		return ErrCircuitOpen
	}

	b.probeInFlight =
		true

	return nil
}

func (b *CircuitBreaker) Success() {
	if b == nil {
		return
	}

	b.mu.Lock()

	defer b.mu.Unlock()

	b.consecutiveFailures =
		0

	b.openedAt =
		time.Time{}

	b.probeInFlight =
		false
}

func (b *CircuitBreaker) Failure(
	now time.Time,
) {
	if b == nil {
		return
	}

	b.mu.Lock()

	defer b.mu.Unlock()

	if !b.openedAt.IsZero() {
		// This was the half-open probe.
		b.openedAt =
			now

		b.probeInFlight =
			false

		b.consecutiveFailures =
			b.config.
				FailureThreshold

		return
	}

	b.consecutiveFailures++

	if b.consecutiveFailures >=
		b.config.
			FailureThreshold {

		b.openedAt =
			now

		b.probeInFlight =
			false
	}
}
