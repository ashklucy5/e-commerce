package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"project.local/commerce-api/internal/jobruntime"
	"project.local/commerce-api/internal/platform/cache"
	"project.local/commerce-api/internal/platform/config"
)

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

	// ---------------------------------------------------------
	// Redis
	// ---------------------------------------------------------

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

	// ---------------------------------------------------------
	// Shared scheduler runtime
	// ---------------------------------------------------------

	runtime, err :=
		jobruntime.NewSchedulerRuntime(
			jobruntime.SchedulerDependencies{
				Redis: redisClient,

				Config: cfg,

				PaymentReconciliationConfig: reconciliationConfig,

				NotificationOutboxConfig: notificationConfig,

				Logger: log.Default(),
			},
		)
	if err != nil {
		log.Fatalf(
			"scheduler runtime startup failed: %v",
			err,
		)
	}

	queueConfig :=
		runtime.QueueConfig()

	log.Printf(
		"schedulers started in queue mode: expiry_interval=%s expiry_batch_size=%d reconciliation_interval=%s reconciliation_batch_size=%d notification_interval=%s notification_batch_size=%d stream=%s",
		cfg.PaymentExpiryInterval,
		cfg.PaymentExpiryBatchSize,
		reconciliationConfig.Interval,
		reconciliationConfig.BatchSize,
		notificationConfig.Interval,
		notificationConfig.BatchSize,
		queueConfig.Stream,
	)

	if err :=
		runtime.Run(
			ctx,
		); err != nil &&
		ctx.Err() == nil {

		log.Fatalf(
			"scheduler stopped with error: %v",
			err,
		)
	}

	log.Printf(
		"scheduler stopped",
	)
}
