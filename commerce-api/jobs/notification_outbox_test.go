package jobs

import (
	"context"
	"testing"
	"time"

	"project.local/commerce-api/internal/notification"
	"project.local/commerce-api/internal/platform/queue"
)

type notificationDispatcherStub struct {
	results []notification.DispatchBatchResult

	err error

	calls int

	batchSizes []int
}

func (s *notificationDispatcherStub) DispatchDue(
	_ context.Context,
	_ time.Time,
	limit int,
) (
	notification.DispatchBatchResult,
	error,
) {
	s.calls++

	s.batchSizes =
		append(
			s.batchSizes,
			limit,
		)

	if s.err != nil {
		return notification.DispatchBatchResult{},
			s.err
	}

	index :=
		s.calls - 1

	if index >=
		len(
			s.results,
		) {

		return notification.DispatchBatchResult{},
			nil
	}

	return s.results[index],
		nil
}

func TestNotificationOutboxHandlerDispatches(
	t *testing.T,
) {
	dispatcher :=
		&notificationDispatcherStub{
			results: []notification.DispatchBatchResult{
				{
					Scanned: 3,

					Sent: 2,

					Skipped: 1,
				},
			},
		}

	handler :=
		NewNotificationOutboxHandler(
			dispatcher,
		)

	err :=
		handler.Handle(
			context.Background(),
			queue.Message{
				Type: NotificationOutboxJobType,

				Payload: []byte(
					`{"batch_size":100}`,
				),
			},
		)
	if err != nil {
		t.Fatalf(
			"handle notification outbox job: %v",
			err,
		)
	}

	if dispatcher.calls != 1 {
		t.Fatalf(
			"expected 1 dispatcher call, got %d",
			dispatcher.calls,
		)
	}

	if len(
		dispatcher.batchSizes,
	) != 1 ||
		dispatcher.batchSizes[0] != 100 {

		t.Fatalf(
			"unexpected batch sizes %#v",
			dispatcher.batchSizes,
		)
	}
}

func TestNotificationOutboxHandlerDrainsBoundedBatches(
	t *testing.T,
) {
	dispatcher :=
		&notificationDispatcherStub{
			results: []notification.DispatchBatchResult{
				{
					Scanned: 100,
				},
				{
					Scanned: 100,
				},
				{
					Scanned: 7,
				},
			},
		}

	handler :=
		NewNotificationOutboxHandler(
			dispatcher,
		)

	err :=
		handler.Handle(
			context.Background(),
			queue.Message{
				Type: NotificationOutboxJobType,

				Payload: []byte(
					`{"batch_size":100}`,
				),
			},
		)
	if err != nil {
		t.Fatalf(
			"handle notification backlog: %v",
			err,
		)
	}

	if dispatcher.calls != 3 {
		t.Fatalf(
			"expected 3 dispatcher batches, got %d",
			dispatcher.calls,
		)
	}
}

func TestNotificationOutboxHandlerRejectsNegativeBatch(
	t *testing.T,
) {
	dispatcher :=
		&notificationDispatcherStub{}

	handler :=
		NewNotificationOutboxHandler(
			dispatcher,
		)

	err :=
		handler.Handle(
			context.Background(),
			queue.Message{
				Type: NotificationOutboxJobType,

				Payload: []byte(
					`{"batch_size":-1}`,
				),
			},
		)

	if err == nil {
		t.Fatal(
			"expected invalid batch size error",
		)
	}

	if !queue.IsPermanent(
		err,
	) {
		t.Fatalf(
			"expected permanent queue error, got %v",
			err,
		)
	}
}

func TestNormalizeNotificationOutboxBatchSize(
	t *testing.T,
) {
	if value :=
		normalizeNotificationOutboxBatchSize(
			0,
		); value !=
		defaultNotificationOutboxBatchSize {

		t.Fatalf(
			"unexpected default batch size %d",
			value,
		)
	}

	if value :=
		normalizeNotificationOutboxBatchSize(
			5000,
		); value !=
		maxNotificationOutboxBatchSize {

		t.Fatalf(
			"unexpected capped batch size %d",
			value,
		)
	}
}
