package queuepipeline_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"project.local/commerce-api/internal/platform/cache"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/queue"
	"project.local/commerce-api/jobs"
	"project.local/commerce-api/scheduler"
)

type fakePaymentExpirer struct {
	called chan int
}

func (f *fakePaymentExpirer) ExpirePendingPayments(
	_ context.Context,
	_ time.Time,
	limit int,
) (int, error) {
	select {
	case f.called <- limit:

	default:
	}

	return 0, nil
}

func TestSchedulerWorkerQueuePipeline(
	t *testing.T,
) {
	if os.Getenv(
		"RUN_QUEUE_PIPELINE_TEST",
	) != "1" {

		t.Skip(
			"set RUN_QUEUE_PIPELINE_TEST=1 to run the Redis queue pipeline integration test",
		)
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			15*time.Second,
		)

	defer cancel()

	cfg, err :=
		config.Load()
	if err != nil {
		t.Fatalf(
			"load config: %v",
			err,
		)
	}

	redisClient, err :=
		cache.NewRedis(
			ctx,
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"open Redis: %v",
			err,
		)
	}

	defer redisClient.Close()

	suffix :=
		fmt.Sprintf(
			"%d",
			time.Now().
				UnixNano(),
		)

	queueConfig :=
		queue.Config{
			Stream: "commerce:e2e:jobs:" +
				suffix,

			Group: "commerce-e2e-workers-" +
				suffix,

			RetrySet: "commerce:e2e:retry:" +
				suffix,

			DeadLetterStream: "commerce:e2e:dead:" +
				suffix,

			DefaultAttempts: 3,

			RetryBaseDelay: 50 *
				time.Millisecond,

			RetryMaxDelay: 250 *
				time.Millisecond,
		}

	t.Cleanup(
		func() {
			cleanupCtx, cleanupCancel :=
				context.WithTimeout(
					context.Background(),
					5*time.Second,
				)

			defer cleanupCancel()

			_ =
				redisClient.Del(
					cleanupCtx,
					queueConfig.Stream,
					queueConfig.RetrySet,
					queueConfig.DeadLetterStream,
				).Err()
		},
	)

	producer, err :=
		queue.NewProducer(
			redisClient,
			queueConfig,
		)
	if err != nil {
		t.Fatalf(
			"create queue producer: %v",
			err,
		)
	}

	expirer :=
		&fakePaymentExpirer{
			called: make(
				chan int,
				1,
			),
		}

	jobHandler :=
		jobs.NewOrderExpirationHandler(
			expirer,
		)

	registry :=
		jobs.NewRegistry()

	if err :=
		registry.Register(
			jobs.OrderExpirationJobType,
			jobHandler.Handle,
		); err != nil {

		t.Fatalf(
			"register order-expiration handler: %v",
			err,
		)
	}

	consumerConfig :=
		queue.DefaultConsumerConfig(
			queueConfig,
		)

	consumerConfig.ConsumerName =
		"commerce-e2e-" +
			suffix

	consumerConfig.Concurrency =
		1

	consumerConfig.Prefetch =
		1

	consumerConfig.Block =
		100 *
			time.Millisecond

	consumerConfig.HandlerTimeout =
		2 *
			time.Second

	consumerConfig.ClaimMinIdle =
		3 *
			time.Second

	consumerConfig.ClaimInterval =
		time.Second

	consumerConfig.RetryPollInterval =
		100 *
			time.Millisecond

	consumerConfig.Logger =
		log.New(
			io.Discard,
			"",
			0,
		)

	consumer, err :=
		queue.NewConsumer(
			redisClient,
			producer,
			consumerConfig,
		)
	if err != nil {
		t.Fatalf(
			"create queue consumer: %v",
			err,
		)
	}

	consumerCtx, consumerCancel :=
		context.WithCancel(
			ctx,
		)

	consumerDone :=
		make(
			chan error,
			1,
		)

	go func() {
		consumerDone <- consumer.Run(
			consumerCtx,
			registry.Handle,
		)
	}()

	enqueuer :=
		jobs.NewOrderExpirationEnqueuer(
			producer,
			2*time.Second,
		)

	runner :=
		scheduler.NewQueue(
			enqueuer,
			time.Hour,
			17,
			log.New(
				io.Discard,
				"",
				0,
			),
		)

	if _, err :=
		runner.RunOnce(
			ctx,
		); err != nil {

		consumerCancel()

		t.Fatalf(
			"scheduler enqueue: %v",
			err,
		)
	}

	select {
	case batchSize :=
		<-expirer.called:

		if batchSize != 17 {
			consumerCancel()

			t.Fatalf(
				"expected worker handler batch size 17, got %d",
				batchSize,
			)
		}

	case <-ctx.Done():

		consumerCancel()

		t.Fatalf(
			"queue pipeline timed out: %v",
			ctx.Err(),
		)
	}

	consumerCancel()

	select {
	case err :=
		<-consumerDone:

		if err != nil {
			t.Fatalf(
				"consumer shutdown: %v",
				err,
			)
		}

	case <-time.After(
		5 *
			time.Second,
	):

		t.Fatal(
			"consumer did not stop after cancellation",
		)
	}
}
