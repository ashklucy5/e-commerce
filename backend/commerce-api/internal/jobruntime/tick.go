package jobruntime

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	defaultTickMaxMessages = 16
	maxTickMessages        = 256
	defaultTickTimeout     = 45 * time.Second
)

type TickConfig struct {
	MaxMessages int

	Timeout time.Duration
}

type TickResult struct {
	JobsProcessed int

	Duration time.Duration
}

type TickRunner struct {
	scheduler *SchedulerRuntime

	worker *Runtime

	config TickConfig
}

func NewTickRunner(
	schedulerRuntime *SchedulerRuntime,
	workerRuntime *Runtime,
	cfg TickConfig,
) (*TickRunner, error) {
	if schedulerRuntime == nil {
		return nil,
			fmt.Errorf(
				"runtime tick requires scheduler runtime",
			)
	}

	if workerRuntime == nil {
		return nil,
			fmt.Errorf(
				"runtime tick requires worker runtime",
			)
	}

	if cfg.MaxMessages == 0 {
		cfg.MaxMessages =
			defaultTickMaxMessages
	}

	if cfg.MaxMessages < 1 ||
		cfg.MaxMessages > maxTickMessages {

		return nil,
			fmt.Errorf(
				"runtime tick max messages must be between 1 and %d",
				maxTickMessages,
			)
	}

	if cfg.Timeout == 0 {
		cfg.Timeout =
			defaultTickTimeout
	}

	if cfg.Timeout <= 0 {
		return nil,
			fmt.Errorf(
				"runtime tick timeout must be greater than zero",
			)
	}

	return &TickRunner{
		scheduler: schedulerRuntime,

		worker: workerRuntime,

		config: cfg,
	}, nil
}

// Run performs one finite background-work cycle.
//
// Scheduling and draining are intentionally separate. If one scheduler
// domain fails, the worker still gets an opportunity to process work
// already present in Redis.
func (r *TickRunner) Run(
	ctx context.Context,
) (
	TickResult,
	error,
) {
	if r == nil ||
		r.scheduler == nil ||
		r.worker == nil {

		return TickResult{},
			fmt.Errorf(
				"runtime tick is not configured",
			)
	}

	if err := ctx.Err(); err != nil {
		return TickResult{},
			err
	}

	startedAt :=
		time.Now()

	tickCtx, cancel :=
		context.WithTimeout(
			ctx,
			r.config.Timeout,
		)

	defer cancel()

	var runErrors []error

	if err :=
		r.scheduler.RunOnce(
			tickCtx,
		); err != nil {

		runErrors =
			append(
				runErrors,
				fmt.Errorf(
					"runtime scheduler pass: %w",
					err,
				),
			)
	}

	processed, err :=
		r.worker.Drain(
			tickCtx,
			r.config.MaxMessages,
		)
	if err != nil {
		runErrors =
			append(
				runErrors,
				fmt.Errorf(
					"runtime queue drain: %w",
					err,
				),
			)
	}

	return TickResult{
			JobsProcessed: processed,

			Duration: time.Since(
				startedAt,
			),
		},
		errors.Join(
			runErrors...,
		)
}
