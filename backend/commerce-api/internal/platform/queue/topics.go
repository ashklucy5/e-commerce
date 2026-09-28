package queue

import (
	"strings"
	"time"
)

const (
	DefaultStream           = "commerce:jobs:v1"
	DefaultGroup            = "commerce-workers-v1"
	DefaultRetrySet         = "commerce:jobs:retry:v1"
	DefaultDeadLetterStream = "commerce:jobs:dead:v1"

	// The live queue is deliberately not length-trimmed.
	//
	// Successfully acknowledged messages are deleted by the consumer.
	// During an outage, preserving queued work is safer than silently
	// trimming old but unprocessed jobs.
	DefaultStreamMaxLen int64 = 0

	// Dead-letter history is diagnostic rather than executable work,
	// so it is safe to cap its retained history.
	DefaultDeadLetterMaxLen int64 = 10000

	DefaultMaxAttempts = 5

	DefaultRetryBaseDelay = time.Second
	DefaultRetryMaxDelay  = time.Minute
)

type Config struct {
	Stream           string
	Group            string
	RetrySet         string
	DeadLetterStream string

	StreamMaxLen     int64
	DeadLetterMaxLen int64
	DefaultAttempts  int

	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}

func DefaultConfig() Config {
	return Config{
		Stream:           DefaultStream,
		Group:            DefaultGroup,
		RetrySet:         DefaultRetrySet,
		DeadLetterStream: DefaultDeadLetterStream,

		StreamMaxLen:     DefaultStreamMaxLen,
		DeadLetterMaxLen: DefaultDeadLetterMaxLen,
		DefaultAttempts:  DefaultMaxAttempts,

		RetryBaseDelay: DefaultRetryBaseDelay,
		RetryMaxDelay:  DefaultRetryMaxDelay,
	}
}

func normalizeConfig(
	cfg Config,
) Config {
	defaults := DefaultConfig()

	cfg.Stream = strings.TrimSpace(
		cfg.Stream,
	)

	if cfg.Stream == "" {
		cfg.Stream = defaults.Stream
	}

	cfg.Group = strings.TrimSpace(
		cfg.Group,
	)

	if cfg.Group == "" {
		cfg.Group = defaults.Group
	}

	cfg.RetrySet = strings.TrimSpace(
		cfg.RetrySet,
	)

	if cfg.RetrySet == "" {
		cfg.RetrySet = defaults.RetrySet
	}

	cfg.DeadLetterStream = strings.TrimSpace(
		cfg.DeadLetterStream,
	)

	if cfg.DeadLetterStream == "" {
		cfg.DeadLetterStream =
			defaults.DeadLetterStream
	}

	if cfg.StreamMaxLen < 0 {
		cfg.StreamMaxLen =
			defaults.StreamMaxLen
	}

	if cfg.DeadLetterMaxLen <= 0 {
		cfg.DeadLetterMaxLen =
			defaults.DeadLetterMaxLen
	}

	if cfg.DefaultAttempts <= 0 {
		cfg.DefaultAttempts =
			defaults.DefaultAttempts
	}

	if cfg.RetryBaseDelay <= 0 {
		cfg.RetryBaseDelay =
			defaults.RetryBaseDelay
	}

	if cfg.RetryMaxDelay <= 0 {
		cfg.RetryMaxDelay =
			defaults.RetryMaxDelay
	}

	if cfg.RetryMaxDelay <
		cfg.RetryBaseDelay {

		cfg.RetryMaxDelay =
			cfg.RetryBaseDelay
	}

	return cfg
}
