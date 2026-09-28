package jobruntime

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/queue"
	"project.local/commerce-api/internal/supportattachment"
	"project.local/commerce-api/jobs"
	"project.local/commerce-api/scheduler"
)

const (
	supportAttachmentRetentionInterval = time.Hour

	/*
		The finite/serverless scheduler can be invoked much more often
		than the traditional one-hour retention loop.

		This Redis gate preserves approximately hourly retention work.
		If an invocation dies after acquiring the gate but before
		enqueueing, the short provisional TTL allows a later tick to
		retry rather than blocking retention for a full hour.
	*/
	retentionProvisionalGateTTL = 5 * time.Minute
)

type SchedulerDependencies struct {
	Redis *redis.Client

	Config config.Config

	PaymentReconciliationConfig config.PaymentReconciliationConfig

	NotificationOutboxConfig config.NotificationOutboxConfig

	Logger *log.Logger
}

type SchedulerRuntime struct {
	redis *redis.Client

	queueConfig queue.Config

	paymentRunner *scheduler.Scheduler

	notificationRunner *scheduler.NotificationOutboxQueue

	retentionRunner *scheduler.SupportAttachmentRetentionQueue

	logger *log.Logger
}

func NewSchedulerRuntime(
	deps SchedulerDependencies,
) (*SchedulerRuntime, error) {
	if deps.Redis == nil {
		return nil,
			fmt.Errorf(
				"scheduler runtime requires Redis",
			)
	}

	logger :=
		deps.Logger

	if logger == nil {
		logger =
			log.Default()
	}

	queueConfig :=
		queue.DefaultConfig()

	producer, err :=
		queue.NewProducer(
			deps.Redis,
			queueConfig,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"create scheduler queue producer: %w",
				err,
			)
	}

	// ---------------------------------------------------------
	// Order / payment expiration
	// ---------------------------------------------------------

	paymentExpiryEnqueuer :=
		jobs.NewOrderExpirationEnqueuer(
			producer,
			schedulerRuntimeDedupeTTL(
				deps.Config.
					PaymentExpiryInterval,
			),
		)

	// ---------------------------------------------------------
	// Payment reconciliation
	// ---------------------------------------------------------

	paymentReconciliationEnqueuer :=
		jobs.NewPaymentReconciliationEnqueuer(
			producer,
			schedulerRuntimeDedupeTTL(
				deps.
					PaymentReconciliationConfig.
					Interval,
			),
		)

	paymentRunner :=
		scheduler.NewQueueWithReconciliation(
			paymentExpiryEnqueuer,
			paymentReconciliationEnqueuer,
			deps.Config.
				PaymentExpiryInterval,
			deps.Config.
				PaymentExpiryBatchSize,
			deps.
				PaymentReconciliationConfig.
				Interval,
			deps.
				PaymentReconciliationConfig.
				BatchSize,
			logger,
		)

	// ---------------------------------------------------------
	// Notification outbox
	// ---------------------------------------------------------

	notificationEnqueuer :=
		jobs.NewNotificationOutboxEnqueuer(
			producer,
			schedulerRuntimeDedupeTTL(
				deps.
					NotificationOutboxConfig.
					Interval,
			),
		)

	notificationRunner :=
		scheduler.NewNotificationOutboxQueue(
			notificationEnqueuer,
			deps.
				NotificationOutboxConfig.
				Interval,
			deps.
				NotificationOutboxConfig.
				BatchSize,
			logger,
		)

	// ---------------------------------------------------------
	// Private support attachment retention
	// ---------------------------------------------------------

	retentionEnqueuer :=
		jobs.NewSupportAttachmentRetentionEnqueuer(
			producer,
			schedulerRuntimeDedupeTTL(
				supportAttachmentRetentionInterval,
			),
		)

	retentionRunner :=
		scheduler.NewSupportAttachmentRetentionQueue(
			retentionEnqueuer,
			supportAttachmentRetentionInterval,
			supportattachment.
				DefaultRetentionBatchSize,
			logger,
		)

	return &SchedulerRuntime{
		redis: deps.Redis,

		queueConfig: queueConfig,

		paymentRunner: paymentRunner,

		notificationRunner: notificationRunner,

		retentionRunner: retentionRunner,

		logger: logger,
	}, nil
}

// Run preserves the existing permanent scheduler execution model.
//
// It starts the three independent scheduler loops and blocks until the
// supplied context is cancelled.
func (r *SchedulerRuntime) Run(
	ctx context.Context,
) error {
	if r == nil ||
		r.paymentRunner == nil ||
		r.notificationRunner == nil ||
		r.retentionRunner == nil {

		return fmt.Errorf(
			"scheduler runtime is not configured",
		)
	}

	var waitGroup sync.WaitGroup

	waitGroup.Add(
		3,
	)

	go func() {
		defer waitGroup.Done()

		r.paymentRunner.Run(
			ctx,
		)
	}()

	go func() {
		defer waitGroup.Done()

		r.notificationRunner.Run(
			ctx,
		)
	}()

	go func() {
		defer waitGroup.Done()

		r.retentionRunner.Run(
			ctx,
		)
	}()

	waitGroup.Wait()

	return nil
}

// RunOnce performs one finite scheduler pass.
//
// This is the serverless execution path. Each scheduler is attempted
// independently so one failed producer does not prevent the remaining
// scheduler domains from being attempted.
//
// Support-attachment retention is additionally guarded so a frequent
// serverless tick does not turn the traditional hourly retention task
// into a task that runs every few minutes.
func (r *SchedulerRuntime) RunOnce(
	ctx context.Context,
) error {
	if r == nil ||
		r.paymentRunner == nil ||
		r.notificationRunner == nil ||
		r.retentionRunner == nil {

		return fmt.Errorf(
			"scheduler runtime is not configured",
		)
	}

	var runErrors []error

	if _, err :=
		r.paymentRunner.RunOnce(
			ctx,
		); err != nil {

		runErrors =
			append(
				runErrors,
				fmt.Errorf(
					"schedule payment expiration: %w",
					err,
				),
			)
	}

	if err :=
		r.paymentRunner.
			RunReconciliationOnce(
				ctx,
			); err != nil {

		runErrors =
			append(
				runErrors,
				fmt.Errorf(
					"schedule payment reconciliation: %w",
					err,
				),
			)
	}

	if err :=
		r.notificationRunner.
			RunOnce(
				ctx,
			); err != nil {

		runErrors =
			append(
				runErrors,
				fmt.Errorf(
					"schedule notification outbox: %w",
					err,
				),
			)
	}

	if err :=
		r.runRetentionIfDue(
			ctx,
		); err != nil {

		runErrors =
			append(
				runErrors,
				fmt.Errorf(
					"schedule support attachment retention: %w",
					err,
				),
			)
	}

	return errors.Join(
		runErrors...,
	)
}

func (r *SchedulerRuntime) runRetentionIfDue(
	ctx context.Context,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	gateKey :=
		r.queueConfig.
			Stream +
			":runtime:support-attachment-retention-gate"

	acquired, err :=
		r.redis.SetNX(
			ctx,
			gateKey,
			"1",
			retentionProvisionalGateTTL,
		).Result()
	if err != nil {
		return fmt.Errorf(
			"acquire retention scheduler gate: %w",
			err,
		)
	}

	if !acquired {
		return nil
	}

	if err :=
		r.retentionRunner.
			RunOnce(
				ctx,
			); err != nil {

		/*
			Do not delete the provisional gate here.

			Leaving the five-minute TTL prevents a failing retention
			job from being hammered on every serverless tick while
			still allowing an automatic retry shortly afterward.
		*/
		return err
	}

	if err :=
		r.redis.Expire(
			ctx,
			gateKey,
			supportAttachmentRetentionInterval,
		).Err(); err != nil {

		return fmt.Errorf(
			"extend retention scheduler gate: %w",
			err,
		)
	}

	return nil
}

func (r *SchedulerRuntime) QueueConfig() queue.Config {
	if r == nil {
		return queue.Config{}
	}

	return r.queueConfig
}

func schedulerRuntimeDedupeTTL(
	interval time.Duration,
) time.Duration {
	dedupeTTL :=
		interval *
			4 /
			5

	if dedupeTTL <
		time.Second {

		dedupeTTL =
			time.Second
	}

	return dedupeTTL
}
