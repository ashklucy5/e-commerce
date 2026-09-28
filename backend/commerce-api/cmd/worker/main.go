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
	"project.local/commerce-api/internal/platform/database"
	"project.local/commerce-api/internal/platform/storage"
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
	// Shared job runtime
	// ---------------------------------------------------------

	runtime, err :=
		jobruntime.New(
			jobruntime.Dependencies{
				DB: db,

				Redis: redisClient,

				Storage: storageGateway,

				Config: cfg,

				WorkerConfig: workerConfig,

				Logger: log.Default(),
			},
		)
	if err != nil {
		log.Fatalf(
			"worker runtime startup failed: %v",
			err,
		)
	}

	queueConfig :=
		runtime.QueueConfig()

	log.Printf(
		"worker started: stream=%s group=%s concurrency=%d prefetch=%d storage=%s",
		queueConfig.Stream,
		queueConfig.Group,
		workerConfig.Concurrency,
		workerConfig.Prefetch,
		runtime.StorageProviderName(),
	)

	if err :=
		runtime.Run(
			ctx,
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
