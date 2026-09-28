package search_test

import (
	"context"
	"os"
	"testing"
	"time"

	"project.local/commerce-api/internal/platform/config"
	"project.local/commerce-api/internal/platform/database"
	"project.local/commerce-api/internal/search"
)

func TestPgvectorProvisioning(
	t *testing.T,
) {
	if os.Getenv(
		"RUN_PGVECTOR_TEST",
	) != "1" {

		t.Skip(
			"set RUN_PGVECTOR_TEST=1 to verify pgvector provisioning",
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

	pool, err :=
		database.NewPostgres(
			ctx,
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"open PostgreSQL: %v",
			err,
		)
	}

	defer pool.Close()

	if err :=
		search.ValidatePgvectorInstalled(
			ctx,
			pool,
		); err != nil {

		t.Fatal(
			err,
		)
	}
}
