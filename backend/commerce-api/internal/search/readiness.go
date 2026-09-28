package search

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type capabilityQueryer interface {
	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
}

func ValidateDatabaseCapabilities(
	ctx context.Context,
	db *pgxpool.Pool,
) error {
	cfg, err :=
		loadSemanticConfigFromEnv()
	if err != nil {
		return err
	}

	return validateDatabaseCapabilities(
		ctx,
		db,
		cfg.Enabled,
	)
}

func ValidatePgvectorInstalled(
	ctx context.Context,
	db *pgxpool.Pool,
) error {
	var installed bool

	if err :=
		db.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1

					FROM pg_extension

					WHERE
						extname = 'vector'
				)
			`,
		).Scan(
			&installed,
		); err != nil {

		return fmt.Errorf(
			"inspect PostgreSQL pgvector extension: %w",
			err,
		)
	}

	if !installed {
		return fmt.Errorf(
			"PostgreSQL extension vector is not installed in the current database",
		)
	}

	return nil
}

func validateDatabaseCapabilities(
	ctx context.Context,
	queryer capabilityQueryer,
	semanticEnabled bool,
) error {
	var trigramInstalled bool
	var vectorInstalled bool

	if err :=
		queryer.QueryRow(
			ctx,
			`
				SELECT
					EXISTS (
						SELECT 1

						FROM pg_extension

						WHERE
							extname = 'pg_trgm'
					),

					EXISTS (
						SELECT 1

						FROM pg_extension

						WHERE
							extname = 'vector'
					)
			`,
		).Scan(
			&trigramInstalled,
			&vectorInstalled,
		); err != nil {

		return fmt.Errorf(
			"inspect PostgreSQL search extensions: %w",
			err,
		)
	}

	if !trigramInstalled {
		return fmt.Errorf(
			"PostgreSQL extension pg_trgm is required for lexical search",
		)
	}

	if semanticEnabled &&
		!vectorInstalled {

		return fmt.Errorf(
			"PostgreSQL extension vector is required when semantic search is enabled",
		)
	}

	return nil
}
