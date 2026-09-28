package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	providerpayments "project.local/commerce-api/integrations/payments"
	"project.local/commerce-api/internal/inventory"
	"project.local/commerce-api/internal/notification"
	"project.local/commerce-api/internal/order"
	"project.local/commerce-api/internal/payment"
	"project.local/commerce-api/internal/platform/cache"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
	"project.local/commerce-api/internal/platform/queue"
	"project.local/commerce-api/internal/platform/storage"
	"project.local/commerce-api/internal/supportattachment"
	"project.local/commerce-api/jobs"
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
			"worker configuration error: %v",
			err,
		)
	}

	workerConfig, err :=
		config.LoadWorkerConfig()
	if err != nil {
		log.Fatalf(
			"worker tuning configuration error: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// PostgreSQL
	// ---------------------------------------------------------

	db, err :=
		database.NewPostgres(
			ctx,
			cfg,
		)
	if err != nil {
		log.Fatalf(
			"worker postgres startup failed: %v",
			err,
		)
	}

	defer db.Close()

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
			"worker redis startup failed: %v",
			err,
		)
	}

	defer func() {
		if err :=
			redisClient.Close(); err != nil {

			log.Printf(
				"worker redis close error: %v",
				err,
			)
		}
	}()

	// ---------------------------------------------------------
	// Private/public object storage
	// ---------------------------------------------------------

	storageConfig, err :=
		config.LoadStorage()
	if err != nil {
		log.Fatalf(
			"worker storage configuration error: %v",
			err,
		)
	}

	storageGateway, err :=
		storage.NewFromConfig(
			storageConfig,
		)
	if err != nil {
		log.Fatalf(
			"worker storage startup failed: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Shared inventory / order services
	// ---------------------------------------------------------

	inventoryRepository :=
		inventory.NewRepository(
			db,
		)

	inventoryService :=
		inventory.NewService(
			inventoryRepository,
		)

	// ---------------------------------------------------------
	// Provider-neutral payment registry
	// ---------------------------------------------------------

	paymentProviders :=
		providerpayments.NewRegistry(
			providerpayments.Enablement{
				BKash: cfg.BKashEnabled,

				Nagad: cfg.NagadEnabled,

				Rocket: cfg.RocketEnabled,

				BankTransfer: cfg.BankTransferEnabled,
			},
		)

	/*
		No fake payment adapters are registered here.

		When genuine provider adapters are added later,
		the same production adapters must be registered
		in both the API and worker registries.
	*/

	orderRepository :=
		order.NewRepository(
			db,
		)

	orderService :=
		order.NewService(
			orderRepository,
			inventoryService,
			order.PaymentMethodConfig{
				CODEnabled: cfg.CODEnabled,

				BKashEnabled: cfg.BKashEnabled,

				NagadEnabled: cfg.NagadEnabled,

				RocketEnabled: cfg.RocketEnabled,

				BankTransferEnabled: cfg.BankTransferEnabled,

				ProviderAvailable: paymentProviders.Available,
			},
		)

	// ---------------------------------------------------------
	// Payment reconciliation
	// ---------------------------------------------------------

	paymentRepository :=
		payment.NewRepository(
			db,
		)

	verifiedPaymentService :=
		payment.NewService(
			paymentRepository,
			orderService,
		)

	paymentReconciliationService :=
		payment.NewReconciliationService(
			paymentRepository,
			paymentProviders,
			verifiedPaymentService,
		)

	// ---------------------------------------------------------
	// Notification outbox
	// ---------------------------------------------------------

	notificationRepository :=
		notification.NewRepository(
			db,
		)

	notificationTemplates, err :=
		notification.
			NewDefaultTemplateRegistry()
	if err != nil {
		log.Fatalf(
			"notification template startup failed: %v",
			err,
		)
	}

	notificationChannels :=
		notification.NewChannelRegistry()

	/*
		Do not register fake email/SMS/messenger providers.

		Until real provider adapters and credentials exist,
		the dispatcher will record provider_unavailable
		rather than pretending a message was delivered.
	*/

	notificationDispatcher :=
		notification.NewDispatcher(
			notificationRepository,
			notificationChannels,
			notificationTemplates,
		)

	// ---------------------------------------------------------
	// Private support attachment retention
	// ---------------------------------------------------------

	supportAttachmentRepository :=
		supportattachment.NewRepository(
			db,
		)

	supportAttachmentRetentionService :=
		supportattachment.NewRetentionService(
			supportAttachmentRepository,
			storageGateway,
		)

	// ---------------------------------------------------------
	// Job registry
	// ---------------------------------------------------------

	registry :=
		jobs.NewRegistry()

	// ---------------------------------------------------------
	// Order expiration job
	// ---------------------------------------------------------

	orderExpirationHandler :=
		jobs.NewOrderExpirationHandler(
			orderService,
		)

	if err :=
		registry.Register(
			jobs.OrderExpirationJobType,
			orderExpirationHandler.Handle,
		); err != nil {

		log.Fatalf(
			"register order expiration job: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Payment reconciliation job
	// ---------------------------------------------------------

	paymentReconciliationHandler :=
		jobs.NewPaymentReconciliationHandler(
			paymentReconciliationService,
		)

	if err :=
		registry.Register(
			jobs.PaymentReconciliationJobType,
			paymentReconciliationHandler.Handle,
		); err != nil {

		log.Fatalf(
			"register payment reconciliation job: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Notification outbox job
	// ---------------------------------------------------------

	notificationOutboxHandler :=
		jobs.NewNotificationOutboxHandler(
			notificationDispatcher,
		)

	if err :=
		registry.Register(
			jobs.NotificationOutboxJobType,
			notificationOutboxHandler.Handle,
		); err != nil {

		log.Fatalf(
			"register notification outbox job: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Private support attachment retention job
	// ---------------------------------------------------------

	supportAttachmentRetentionHandler :=
		jobs.NewSupportAttachmentRetentionHandler(
			supportAttachmentRetentionService,
		)

	if err :=
		registry.Register(
			jobs.SupportAttachmentRetentionJobType,
			supportAttachmentRetentionHandler.Handle,
		); err != nil {

		log.Fatalf(
			"register support attachment retention job: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Queue worker
	// ---------------------------------------------------------

	queueConfig :=
		queue.DefaultConfig()

	producer, err :=
		queue.NewProducer(
			redisClient,
			queueConfig,
		)
	if err != nil {
		log.Fatalf(
			"worker queue producer startup failed: %v",
			err,
		)
	}

	consumerConfig :=
		queue.DefaultConsumerConfig(
			queueConfig,
		)

	consumerConfig.Concurrency =
		workerConfig.Concurrency

	consumerConfig.Prefetch =
		workerConfig.Prefetch

	consumerConfig.Block =
		workerConfig.Block

	consumerConfig.HandlerTimeout =
		workerConfig.HandlerTimeout

	consumerConfig.ClaimMinIdle =
		workerConfig.ClaimMinIdle

	consumerConfig.ClaimInterval =
		workerConfig.ClaimInterval

	consumerConfig.RetryPollInterval =
		workerConfig.RetryPollInterval

	consumerConfig.Logger =
		log.Default()

	consumer, err :=
		queue.NewConsumer(
			redisClient,
			producer,
			consumerConfig,
		)
	if err != nil {
		log.Fatalf(
			"worker queue consumer startup failed: %v",
			err,
		)
	}

	log.Printf(
		"worker started: stream=%s group=%s concurrency=%d prefetch=%d storage=%s",
		queueConfig.Stream,
		queueConfig.Group,
		consumerConfig.Concurrency,
		consumerConfig.Prefetch,
		storageGateway.ProviderName(),
	)

	if err :=
		consumer.Run(
			ctx,
			registry.Handle,
		); err != nil &&
		ctx.Err() == nil {

		log.Fatalf(
			"worker stopped with error: %v",
			err,
		)
	}

	log.Printf(
		"worker stopped",
	)
}
