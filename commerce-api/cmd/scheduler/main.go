package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"project.local/commerce-api/internal/platform/cache"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/queue"
	"project.local/commerce-api/internal/supportattachment"
	"project.local/commerce-api/jobs"
	"project.local/commerce-api/scheduler"
)

const supportAttachmentRetentionInterval = time.Hour

func main() {
	ctx, stop :=
		signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)

	defer stop()

	cfg, err :=
		config.Load()
	if err != nil {
		log.Fatalf(
			"scheduler configuration error: %v",
			err,
		)
	}

	reconciliationConfig, err :=
		config.LoadPaymentReconciliationConfig()
	if err != nil {
		log.Fatalf(
			"payment reconciliation scheduler configuration error: %v",
			err,
		)
	}

	notificationConfig, err :=
		config.LoadNotificationOutboxConfig()
	if err != nil {
		log.Fatalf(
			"notification outbox scheduler configuration error: %v",
			err,
		)
	}

	redisClient, err :=
		cache.NewRedis(
			ctx,
			cfg,
		)
	if err != nil {
		log.Fatalf(
			"scheduler redis startup failed: %v",
			err,
		)
	}

	defer func() {
		if err :=
			redisClient.Close(); err != nil {

			log.Printf(
				"scheduler redis close error: %v",
				err,
			)
		}
	}()

	queueConfig :=
		queue.DefaultConfig()

	producer, err :=
		queue.NewProducer(
			redisClient,
			queueConfig,
		)
	if err != nil {
		log.Fatalf(
			"scheduler queue startup failed: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Order/payment expiry
	// ---------------------------------------------------------

	paymentExpiryDedupeTTL :=
		schedulerDedupeTTL(
			cfg.PaymentExpiryInterval,
		)

	paymentExpiryEnqueuer :=
		jobs.NewOrderExpirationEnqueuer(
			producer,
			paymentExpiryDedupeTTL,
		)

	// ---------------------------------------------------------
	// Payment reconciliation
	// ---------------------------------------------------------

	paymentReconciliationDedupeTTL :=
		schedulerDedupeTTL(
			reconciliationConfig.Interval,
		)

	paymentReconciliationEnqueuer :=
		jobs.NewPaymentReconciliationEnqueuer(
			producer,
			paymentReconciliationDedupeTTL,
		)

	paymentRunner :=
		scheduler.NewQueueWithReconciliation(
			paymentExpiryEnqueuer,
			paymentReconciliationEnqueuer,
			cfg.PaymentExpiryInterval,
			cfg.PaymentExpiryBatchSize,
			reconciliationConfig.Interval,
			reconciliationConfig.BatchSize,
			log.Default(),
		)

	// ---------------------------------------------------------
	// Notification outbox
	// ---------------------------------------------------------

	notificationDedupeTTL :=
		schedulerDedupeTTL(
			notificationConfig.Interval,
		)

	notificationEnqueuer :=
		jobs.NewNotificationOutboxEnqueuer(
			producer,
			notificationDedupeTTL,
		)

	notificationRunner :=
		scheduler.NewNotificationOutboxQueue(
			notificationEnqueuer,
			notificationConfig.Interval,
			notificationConfig.BatchSize,
			log.Default(),
		)

	// ---------------------------------------------------------
	// Private support attachment retention
	// ---------------------------------------------------------

	retentionDedupeTTL :=
		schedulerDedupeTTL(
			supportAttachmentRetentionInterval,
		)

	retentionEnqueuer :=
		jobs.NewSupportAttachmentRetentionEnqueuer(
			producer,
			retentionDedupeTTL,
		)

	retentionRunner :=
		scheduler.NewSupportAttachmentRetentionQueue(
			retentionEnqueuer,
			supportAttachmentRetentionInterval,
			supportattachment.
				DefaultRetentionBatchSize,
			log.Default(),
		)

	log.Printf(
		"schedulers started in queue mode: expiry_interval=%s expiry_batch_size=%d reconciliation_interval=%s reconciliation_batch_size=%d notification_interval=%s notification_batch_size=%d support_attachment_retention_interval=%s support_attachment_retention_batch_size=%d stream=%s",
		cfg.PaymentExpiryInterval,
		cfg.PaymentExpiryBatchSize,
		reconciliationConfig.Interval,
		reconciliationConfig.BatchSize,
		notificationConfig.Interval,
		notificationConfig.BatchSize,
		supportAttachmentRetentionInterval,
		supportattachment.
			DefaultRetentionBatchSize,
		queueConfig.Stream,
	)

	/*
		Each scheduler is an independent loop.

		A temporary failure in one domain does not stop payment,
		notification, or support-attachment scheduling in the
		other domains.
	*/
	var waitGroup sync.WaitGroup

	waitGroup.Add(
		3,
	)

	go func() {
		defer waitGroup.Done()

		paymentRunner.Run(
			ctx,
		)
	}()

	go func() {
		defer waitGroup.Done()

		notificationRunner.Run(
			ctx,
		)
	}()

	go func() {
		defer waitGroup.Done()

		retentionRunner.Run(
			ctx,
		)
	}()

	waitGroup.Wait()

	log.Printf(
		"scheduler stopped",
	)
}

func schedulerDedupeTTL(
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
