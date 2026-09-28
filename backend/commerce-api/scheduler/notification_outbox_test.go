package scheduler

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"
)

type notificationOutboxEnqueuerStub struct {
	calls int

	batchSizes []int

	err error
}

func (
	s *notificationOutboxEnqueuerStub,
) Enqueue(
	_ context.Context,
	batchSize int,
) error {
	s.calls++

	s.batchSizes =
		append(
			s.batchSizes,
			batchSize,
		)

	return s.err
}

func TestNotificationOutboxQueueEnqueue(
	t *testing.T,
) {
	enqueuer :=
		&notificationOutboxEnqueuerStub{}

	runner :=
		NewNotificationOutboxQueue(
			enqueuer,
			time.Second,
			100,
			log.New(
				io.Discard,
				"",
				0,
			),
		)

	runner.enqueue(
		context.Background(),
	)

	if enqueuer.calls != 1 {
		t.Fatalf(
			"expected one enqueue call, got %d",
			enqueuer.calls,
		)
	}

	if len(
		enqueuer.batchSizes,
	) != 1 ||
		enqueuer.batchSizes[0] != 100 {

		t.Fatalf(
			"unexpected batch sizes %#v",
			enqueuer.batchSizes,
		)
	}
}

func TestNotificationOutboxQueueStopsForCancelledContext(
	t *testing.T,
) {
	enqueuer :=
		&notificationOutboxEnqueuerStub{}

	runner :=
		NewNotificationOutboxQueue(
			enqueuer,
			time.Second,
			100,
			log.New(
				io.Discard,
				"",
				0,
			),
		)

	ctx, cancel :=
		context.WithCancel(
			context.Background(),
		)

	cancel()

	runner.Run(
		ctx,
	)

	if enqueuer.calls != 0 {
		t.Fatalf(
			"cancelled scheduler unexpectedly enqueued %d jobs",
			enqueuer.calls,
		)
	}
}

func TestNotificationOutboxQueueContinuesAfterEnqueueError(
	t *testing.T,
) {
	enqueuer :=
		&notificationOutboxEnqueuerStub{
			err: errors.New(
				"redis temporarily unavailable",
			),
		}

	runner :=
		NewNotificationOutboxQueue(
			enqueuer,
			time.Second,
			100,
			log.New(
				io.Discard,
				"",
				0,
			),
		)

	runner.enqueue(
		context.Background(),
	)

	if enqueuer.calls != 1 {
		t.Fatalf(
			"expected one attempted enqueue, got %d",
			enqueuer.calls,
		)
	}
}
