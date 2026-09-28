package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	admincore "project.local/commerce-api/internal/admin"
	"project.local/commerce-api/internal/order"
	"project.local/commerce-api/internal/platform/cache"
	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
	"project.local/commerce-api/internal/platform/router"
	"project.local/commerce-api/internal/platform/storage"
)

func main() {
	cfg, err :=
		config.Load()
	if err != nil {
		log.Fatalf(
			"configuration error: %v",
			err,
		)
	}

	if err :=
		order.ValidateInvoiceRuntimeConfig(); err != nil {

		log.Fatalf(
			"invoice configuration error: %v",
			err,
		)
	}

	adminSecurity, err :=
		config.LoadAdminSecurityConfig(
			cfg.AppEnv,
		)
	if err != nil {
		log.Fatalf(
			"Admin security configuration error: %v",
			err,
		)
	}

	appAllowedOrigins, err :=
		config.LoadAppAllowedOrigins(
			cfg.AppEnv,
		)
	if err != nil {
		log.Fatalf(
			"storefront CORS configuration error: %v",
			err,
		)
	}

	metricsConfig, err :=
		config.LoadMetricsConfig(
			cfg.AppEnv,
		)
	if err != nil {
		log.Fatalf(
			"metrics configuration error: %v",
			err,
		)
	}

	ctx :=
		context.Background()

	appliedMigrations, err :=
		database.ApplyMigrations(
			ctx,
			cfg,
		)
	if err != nil {
		log.Fatalf(
			"database migration startup failed: %v",
			err,
		)
	}

	if appliedMigrations > 0 {
		log.Printf(
			"database migrations applied: %d",
			appliedMigrations,
		)
	} else {
		log.Println(
			"database migrations current",
		)
	}

	db, err :=
		database.NewPostgres(
			ctx,
			cfg,
		)
	if err != nil {
		log.Fatalf(
			"postgres startup failed: %v",
			err,
		)
	}

	defer db.Close()

	/*
		Synchronize built-in Admin permissions and the
		admin_superuser system role after PostgreSQL is ready.

		This keeps newly added system permissions, such as:

			admin.finance.read
			admin.finance.manage

		present in the database and automatically assigned to
		the Admin Superuser role without requiring another
		manual bootstrap operation.
	*/
	if err :=
		admincore.SyncSystemAuthorization(
			ctx,
			db,
		); err != nil {

		log.Fatalf(
			"Admin authorization synchronization failed: %v",
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
			"redis startup failed: %v",
			err,
		)
	}

	defer func() {
		if err :=
			redisClient.Close(); err != nil {

			log.Printf(
				"redis close error: %v",
				err,
			)
		}
	}()

	storageConfig, err :=
		config.LoadStorage()
	if err != nil {
		log.Fatalf(
			"storage configuration error: %v",
			err,
		)
	}

	storageGateway, err :=
		storage.NewFromConfig(
			storageConfig,
		)
	if err != nil {
		log.Fatalf(
			"storage startup failed: %v",
			err,
		)
	}

	engine :=
		router.New(
			router.Dependencies{
				DB: db,

				Redis: redisClient,

				Storage: storageGateway,

				AppEnv: cfg.AppEnv,

				AdminSecurity: adminSecurity,

				AppAllowedOrigins: appAllowedOrigins,

				TrustedProxies: cfg.TrustedProxies,

				Metrics: metricsConfig,

				CODEnabled: cfg.CODEnabled,

				BKashEnabled: cfg.BKashEnabled,

				NagadEnabled: cfg.NagadEnabled,

				RocketEnabled: cfg.RocketEnabled,

				BankTransferEnabled: cfg.BankTransferEnabled,
			},
		)

	server :=
		&http.Server{
			Addr: cfg.HTTPAddr,

			Handler: engine,

			ReadHeaderTimeout: 5 * time.Second,

			ReadTimeout: 15 * time.Second,

			WriteTimeout: 30 * time.Second,

			IdleTimeout: 60 * time.Second,
		}

	serverErrors :=
		make(
			chan error,
			1,
		)

	go func() {
		log.Printf(
			"commerce API starting on %s (%s)",
			cfg.HTTPAddr,
			cfg.AppEnv,
		)

		err :=
			server.ListenAndServe()

		if err != nil &&
			!errors.Is(
				err,
				http.ErrServerClosed,
			) {

			serverErrors <- err
		}
	}()

	signalCtx, stop :=
		signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)

	defer stop()

	select {
	case err :=
		<-serverErrors:

		log.Fatalf(
			"HTTP server failed: %v",
			err,
		)

	case <-signalCtx.Done():

		log.Println(
			"shutdown signal received",
		)
	}

	shutdownCtx, cancel :=
		context.WithTimeout(
			context.Background(),
			cfg.ShutdownTimeout,
		)

	defer cancel()

	if err :=
		server.Shutdown(
			shutdownCtx,
		); err != nil {

		log.Printf(
			"graceful shutdown failed: %v",
			err,
		)
	}

	log.Println(
		"commerce API stopped",
	)
}
