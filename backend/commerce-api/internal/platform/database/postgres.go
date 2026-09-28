package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"project.local/commerce-api/internal/platform/config"
)

func NewPostgres(
	ctx context.Context,
	cfg config.Config,
) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(
		cfg.PostgresURL(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"parse postgres config: %w",
			err,
		)
	}

	poolConfig.MaxConns = 20
	poolConfig.MinConns = 2

	poolConfig.MaxConnLifetime =
		30 * time.Minute

	poolConfig.MaxConnIdleTime =
		5 * time.Minute

	pool, err := pgxpool.NewWithConfig(
		ctx,
		poolConfig,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create postgres pool: %w",
			err,
		)
	}

	pingCtx, cancel :=
		context.WithTimeout(
			ctx,
			5*time.Second,
		)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()

		return nil, fmt.Errorf(
			"postgres ping failed: %w",
			err,
		)
	}

	return pool, nil
}
