package jobruntime

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	providerpayments "project.local/commerce-api/integrations/payments"
	"project.local/commerce-api/internal/inventory"
	"project.local/commerce-api/internal/notification"
	"project.local/commerce-api/internal/order"
	"project.local/commerce-api/internal/payment"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/queue"
	"project.local/commerce-api/internal/platform/storage"
	"project.local/commerce-api/internal/supportattachment"
	"project.local/commerce-api/jobs"
)

type Dependencies struct {
	DB *pgxpool.Pool

	Redis *redis.Client

	Storage *storage.Gateway

	Config config.Config

	WorkerConfig config.WorkerConfig

	Logger *log.Logger
}

type Runtime struct {
	consumer *queue.Consumer

	registry *jobs.Registry

	queueConfig queue.Config

	storageProvider string
}

func New(
	deps Dependencies,
) (*Runtime, error) {
	if deps.DB == nil {
		return nil,
			fmt.Errorf(
				"job runtime requires PostgreSQL",
			)
	}

	if deps.Redis == nil {
		return nil,
			fmt.Errorf(
				"job runtime requires Redis",
			)
	}

	if deps.Storage == nil {
		return nil,
			fmt.Errorf(
				"job runtime requires storage gateway",
			)
	}

	logger :=
		deps.Logger

	if logger == nil {
		logger =
			log.Default()
	}

	// ---------------------------------------------------------
	// Shared inventory / order services
	// ---------------------------------------------------------

	inventoryRepository :=
		inventory.NewRepository(
			deps.DB,
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
				BKash: deps.Config.
					BKashEnabled,

				Nagad: deps.Config.
					NagadEnabled,

				Rocket: deps.Config.
					RocketEnabled,

				BankTransfer: deps.Config.
					BankTransferEnabled,
			},
		)

	/*
		Do not register fake payment adapters.

		When genuine provider adapters are added later, the same
		production adapters must be registered here so both the
		continuous worker and bounded runtime use identical
		provider behavior.
	*/

	orderRepository :=
		order.NewRepository(
			deps.DB,
		)

	orderService :=
		order.NewService(
			orderRepository,
			inventoryService,
			order.PaymentMethodConfig{
				CODEnabled: deps.Config.
					CODEnabled,

				BKashEnabled: deps.Config.
					BKashEnabled,

				NagadEnabled: deps.Config.
					NagadEnabled,

				RocketEnabled: deps.Config.
					RocketEnabled,

				BankTransferEnabled: deps.Config.
					BankTransferEnabled,

				ProviderAvailable: paymentProviders.Available,
			},
		)

	// ---------------------------------------------------------
	// Payment reconciliation
	// ---------------------------------------------------------

	paymentRepository :=
		payment.NewRepository(
			deps.DB,
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
			deps.DB,
		)

	notificationTemplates, err :=
		notification.
			NewDefaultTemplateRegistry()
	if err != nil {
		return nil,
			fmt.Errorf(
				"create notification template registry: %w",
				err,
			)
	}

	notificationChannels :=
		notification.NewChannelRegistry()

	/*
		Do not register fake email/SMS/messenger providers.

		Until real provider adapters and credentials exist, the
		dispatcher records provider_unavailable rather than
		pretending delivery succeeded.
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
			deps.DB,
		)

	supportAttachmentRetentionService :=
		supportattachment.NewRetentionService(
			supportAttachmentRepository,
			deps.Storage,
		)

	// ---------------------------------------------------------
	// Job registry
	// ---------------------------------------------------------

	registry :=
		jobs.NewRegistry()

	orderExpirationHandler :=
		jobs.NewOrderExpirationHandler(
			orderService,
		)

	if err :=
		registry.Register(
			jobs.OrderExpirationJobType,
			orderExpirationHandler.Handle,
		); err != nil {

		return nil,
			fmt.Errorf(
				"register order expiration job: %w",
				err,
			)
	}

	paymentReconciliationHandler :=
		jobs.NewPaymentReconciliationHandler(
			paymentReconciliationService,
		)

	if err :=
		registry.Register(
			jobs.PaymentReconciliationJobType,
			paymentReconciliationHandler.Handle,
		); err != nil {

		return nil,
			fmt.Errorf(
				"register payment reconciliation job: %w",
				err,
			)
	}

	notificationOutboxHandler :=
		jobs.NewNotificationOutboxHandler(
			notificationDispatcher,
		)

	if err :=
		registry.Register(
			jobs.NotificationOutboxJobType,
			notificationOutboxHandler.Handle,
		); err != nil {

		return nil,
			fmt.Errorf(
				"register notification outbox job: %w",
				err,
			)
	}

	supportAttachmentRetentionHandler :=
		jobs.NewSupportAttachmentRetentionHandler(
			supportAttachmentRetentionService,
		)

	if err :=
		registry.Register(
			jobs.SupportAttachmentRetentionJobType,
			supportAttachmentRetentionHandler.Handle,
		); err != nil {

		return nil,
			fmt.Errorf(
				"register support attachment retention job: %w",
				err,
			)
	}

	// ---------------------------------------------------------
	// Queue producer / consumer
	// ---------------------------------------------------------

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
				"create queue producer: %w",
				err,
			)
	}

	consumerConfig :=
		queue.DefaultConsumerConfig(
			queueConfig,
		)

	consumerConfig.Concurrency =
		deps.WorkerConfig.
			Concurrency

	consumerConfig.Prefetch =
		deps.WorkerConfig.
			Prefetch

	consumerConfig.Block =
		deps.WorkerConfig.
			Block

	consumerConfig.HandlerTimeout =
		deps.WorkerConfig.
			HandlerTimeout

	consumerConfig.ClaimMinIdle =
		deps.WorkerConfig.
			ClaimMinIdle

	consumerConfig.ClaimInterval =
		deps.WorkerConfig.
			ClaimInterval

	consumerConfig.RetryPollInterval =
		deps.WorkerConfig.
			RetryPollInterval

	consumerConfig.Logger =
		logger

	consumer, err :=
		queue.NewConsumer(
			deps.Redis,
			producer,
			consumerConfig,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"create queue consumer: %w",
				err,
			)
	}

	return &Runtime{
		consumer: consumer,

		registry: registry,

		queueConfig: queueConfig,

		storageProvider: deps.Storage.
			ProviderName(),
	}, nil
}

// Run preserves the existing permanent-worker execution model.
func (r *Runtime) Run(
	ctx context.Context,
) error {
	if r == nil ||
		r.consumer == nil ||
		r.registry == nil {

		return fmt.Errorf(
			"job runtime is not configured",
		)
	}

	return r.consumer.Run(
		ctx,
		r.registry.Handle,
	)
}

// Drain processes at most maxMessages currently available jobs and returns.
// It is the finite execution path used by the serverless runtime endpoint.
func (r *Runtime) Drain(
	ctx context.Context,
	maxMessages int,
) (int, error) {
	if r == nil ||
		r.consumer == nil ||
		r.registry == nil {

		return 0,
			fmt.Errorf(
				"job runtime is not configured",
			)
	}

	return r.consumer.Drain(
		ctx,
		r.registry.Handle,
		maxMessages,
	)
}

func (r *Runtime) QueueConfig() queue.Config {
	if r == nil {
		return queue.Config{}
	}

	return r.queueConfig
}

func (r *Runtime) StorageProviderName() string {
	if r == nil {
		return ""
	}

	return r.storageProvider
}
