package database

import (
	"context"
	"os"
	"strings"
	"testing"

	"ariga.io/atlas/sql/migrate"

	"project.local/commerce-api/internal/platform/config"
)

func TestEmbeddedMigrationsValidate(
	t *testing.T,
) {
	dir, err :=
		loadEmbeddedMigrationDir()
	if err != nil {
		t.Fatalf(
			"load embedded migrations: %v",
			err,
		)
	}

	defer dir.Close()

	if err :=
		migrate.Validate(
			dir,
		); err != nil {

		t.Fatalf(
			"validate embedded migrations: %v",
			err,
		)
	}

	files, err :=
		dir.Files()
	if err != nil {
		t.Fatalf(
			"list embedded migrations: %v",
			err,
		)
	}

	if len(
		files,
	) == 0 {
		t.Fatal(
			"expected embedded migration files",
		)
	}

	for _, file := range files {

		if _, err :=
			migrationFileTxMode(
				file,
			); err != nil {

			t.Fatalf(
				"migration %s has unsupported transaction mode: %v",
				file.Name(),
				err,
			)
		}
	}
}

func TestMigrationPostgresURLUsesPublicSearchPath(
	t *testing.T,
) {
	dsn, err :=
		migrationPostgresURL(
			config.Config{
				PostgresHost: "localhost",

				PostgresPort: 5432,

				PostgresDB: "commerce",

				PostgresUser: "commerce",

				PostgresPassword: "secret",

				PostgresSSLMode: "disable",
			},
		)
	if err != nil {
		t.Fatalf(
			"build migration URL: %v",
			err,
		)
	}

	if !strings.Contains(
		dsn,
		"search_path=public",
	) {
		t.Fatalf(
			"migration URL must use public search_path: %s",
			dsn,
		)
	}
}

func TestApplyMigrationsLive(
	t *testing.T,
) {
	if os.Getenv(
		"RUN_DB_MIGRATION_TEST",
	) != "1" {
		t.Skip(
			"set RUN_DB_MIGRATION_TEST=1 to run against the configured development database",
		)
	}

	cfg, err :=
		config.Load()
	if err != nil {
		t.Fatalf(
			"load config: %v",
			err,
		)
	}

	first, err :=
		ApplyMigrations(
			context.Background(),
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"first migration pass: %v",
			err,
		)
	}

	second, err :=
		ApplyMigrations(
			context.Background(),
			cfg,
		)
	if err != nil {
		t.Fatalf(
			"second migration pass: %v",
			err,
		)
	}

	if second != 0 {
		t.Fatalf(
			"expected idempotent second migration pass, applied %d files",
			second,
		)
	}

	t.Logf(
		"migration startup check passed; first pass applied %d file(s), second pass applied 0",
		first,
	)
}
