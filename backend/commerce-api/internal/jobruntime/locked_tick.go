package jobruntime

import (
	"context"
	"fmt"
	"time"
)

const runtimeTickReleaseTimeout = 5 * time.Second

type LockedTickResult struct {
	Acquired bool

	Tick TickResult
}

type LockedTickRunner struct {
	runner *TickRunner

	lock *TickLock
}

func NewLockedTickRunner(
	runner *TickRunner,
	lock *TickLock,
) (*LockedTickRunner, error) {
	if runner == nil {
		return nil,
			fmt.Errorf(
				"locked runtime tick requires tick runner",
			)
	}

	if lock == nil {
		return nil,
			fmt.Errorf(
				"locked runtime tick requires tick lock",
			)
	}

	return &LockedTickRunner{
		runner: runner,

		lock: lock,
	}, nil
}

func (r *LockedTickRunner) Run(
	ctx context.Context,
) (
	LockedTickResult,
	error,
) {
	if r == nil ||
		r.runner == nil ||
		r.lock == nil {

		return LockedTickResult{},
			fmt.Errorf(
				"locked runtime tick is not configured",
			)
	}

	lease, acquired, err :=
		r.lock.Acquire(
			ctx,
		)
	if err != nil {
		return LockedTickResult{},
			err
	}

	if !acquired {
		return LockedTickResult{
			Acquired: false,
		}, nil
	}

	/*
		Release uses its own short context.

		The runtime work context may already have reached its timeout,
		but we should still attempt to remove our lock immediately
		instead of unnecessarily waiting for the Redis TTL.
	*/
	defer func() {
		releaseCtx, cancel :=
			context.WithTimeout(
				context.Background(),
				runtimeTickReleaseTimeout,
			)

		defer cancel()

		_ =
			lease.Release(
				releaseCtx,
			)
	}()

	result, err :=
		r.runner.Run(
			ctx,
		)

	return LockedTickResult{
			Acquired: true,

			Tick: result,
		},
		err
}
