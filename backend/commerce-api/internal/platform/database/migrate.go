package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"
	"time"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"

	dbmigrations "project.local/commerce-api/db/migrations"
	"project.local/commerce-api/internal/platform/config"
)

const (
	atlasRevisionSchema = "public"

	atlasRevisionTable = "atlas_schema_revisions"

	migrationLockName = "atlas_migrate_execute"

	migrationLockWait = 2 * time.Minute
)

// ApplyMigrations validates and applies all pending Atlas migrations before the
// API serves traffic. A PostgreSQL advisory lock serializes concurrent starters.
func ApplyMigrations(
	ctx context.Context,
	cfg config.Config,
) (int, error) {
	dir, err :=
		loadEmbeddedMigrationDir()
	if err != nil {
		return 0, err
	}

	defer dir.Close()

	if err :=
		migrate.Validate(
			dir,
		); err != nil {

		return 0,
			fmt.Errorf(
				"validate embedded migration directory: %w",
				err,
			)
	}

	dsn, err :=
		migrationPostgresURL(
			cfg,
		)
	if err != nil {
		return 0, err
	}

	db, err :=
		sql.Open(
			"pgx",
			dsn,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"open migration database: %w",
				err,
			)
	}

	defer db.Close()

	db.SetMaxOpenConns(
		1,
	)

	db.SetMaxIdleConns(
		1,
	)

	conn, err :=
		db.Conn(
			ctx,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"acquire migration database connection: %w",
				err,
			)
	}

	defer conn.Close()

	driver, err :=
		postgres.Open(
			conn,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"open Atlas PostgreSQL driver: %w",
				err,
			)
	}

	unlock, err :=
		driver.Lock(
			ctx,
			migrationLockName,
			migrationLockWait,
		)
	if err != nil {
		return 0,
			fmt.Errorf(
				"acquire migration advisory lock: %w",
				err,
			)
	}

	defer func() {
		_ =
			unlock()
	}()

	revisions :=
		&atlasRevisionStore{
			db: conn,
		}

	if err :=
		revisions.ensure(
			ctx,
		); err != nil {

		return 0, err
	}

	executor, err :=
		newAtlasExecutor(
			driver,
			dir,
			revisions,
		)
	if err != nil {
		return 0, err
	}

	pending, err :=
		executor.Pending(
			ctx,
		)

	if errors.Is(
		err,
		migrate.ErrNoPendingFiles,
	) {
		return 0, nil
	}

	if err != nil {
		return 0,
			fmt.Errorf(
				"resolve pending migrations: %w",
				err,
			)
	}

	for index, file := range pending {

		if err :=
			executeMigrationFile(
				ctx,
				conn,
				dir,
				executor,
				file,
			); err != nil {

			return index,
				fmt.Errorf(
					"apply migration %s: %w",
					file.Name(),
					err,
				)
		}
	}

	return len(
			pending,
		),
		nil
}

func loadEmbeddedMigrationDir() (
	*migrate.MemDir,
	error,
) {
	dir :=
		migrate.OpenMemDir(
			"db/migrations",
		)

	entries, err :=
		fs.ReadDir(
			dbmigrations.Files,
			".",
		)
	if err != nil {
		_ =
			dir.Close()

		return nil,
			fmt.Errorf(
				"read embedded migration directory: %w",
				err,
			)
	}

	for _, entry := range entries {

		if entry.IsDir() {
			continue
		}

		data, err :=
			fs.ReadFile(
				dbmigrations.Files,
				entry.Name(),
			)
		if err != nil {
			_ =
				dir.Close()

			return nil,
				fmt.Errorf(
					"read embedded migration file %s: %w",
					entry.Name(),
					err,
				)
		}

		if err :=
			dir.WriteFile(
				entry.Name(),
				data,
			); err != nil {

			_ =
				dir.Close()

			return nil,
				fmt.Errorf(
					"load embedded migration file %s: %w",
					entry.Name(),
					err,
				)
		}
	}

	return dir, nil
}

func migrationPostgresURL(
	cfg config.Config,
) (string, error) {
	u, err :=
		url.Parse(
			cfg.PostgresURL(),
		)
	if err != nil {
		return "",
			fmt.Errorf(
				"parse migration postgres URL: %w",
				err,
			)
	}

	query :=
		u.Query()

	query.Set(
		"search_path",
		atlasRevisionSchema,
	)

	u.RawQuery =
		query.Encode()

	return u.String(), nil
}

func newAtlasExecutor(
	driver migrate.Driver,
	dir migrate.Dir,
	revisions migrate.RevisionReadWriter,
) (
	*migrate.Executor,
	error,
) {
	executor, err :=
		migrate.NewExecutor(
			driver,
			dir,
			revisions,
			migrate.WithExecOrder(
				migrate.ExecOrderLinear,
			),
			migrate.WithOperatorVersion(
				"commerce-api",
			),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"create Atlas migration executor: %w",
				err,
			)
	}

	return executor, nil
}

func executeMigrationFile(
	ctx context.Context,
	conn *sql.Conn,
	dir migrate.Dir,
	baseExecutor *migrate.Executor,
	file migrate.File,
) error {
	mode, err :=
		migrationFileTxMode(
			file,
		)
	if err != nil {
		return err
	}

	if mode == "none" {
		return baseExecutor.Execute(
			ctx,
			file,
		)
	}

	tx, err :=
		conn.BeginTx(
			ctx,
			nil,
		)
	if err != nil {
		return fmt.Errorf(
			"begin migration transaction: %w",
			err,
		)
	}

	committed :=
		false

	defer func() {
		if !committed {
			_ =
				tx.Rollback()
		}
	}()

	txDriver, err :=
		postgres.Open(
			tx,
		)
	if err != nil {
		return fmt.Errorf(
			"open transactional Atlas driver: %w",
			err,
		)
	}

	txExecutor, err :=
		newAtlasExecutor(
			txDriver,
			dir,
			&atlasRevisionStore{
				db: tx,
			},
		)
	if err != nil {
		return err
	}

	if err :=
		txExecutor.Execute(
			ctx,
			file,
		); err != nil {

		if rollbackErr :=
			tx.Rollback(); rollbackErr != nil &&
			!errors.Is(
				rollbackErr,
				sql.ErrTxDone,
			) {

			return errors.Join(
				err,
				fmt.Errorf(
					"rollback migration transaction: %w",
					rollbackErr,
				),
			)
		}

		return err
	}

	if err :=
		tx.Commit(); err != nil {

		return fmt.Errorf(
			"commit migration transaction: %w",
			err,
		)
	}

	committed =
		true

	return nil
}

type fileDirectiveReader interface {
	Directive(
		name string,
	) []string
}

func migrationFileTxMode(
	file migrate.File,
) (string, error) {
	mode :=
		"file"

	if directives, ok :=
		file.(fileDirectiveReader); ok {

		values :=
			directives.Directive(
				"txmode",
			)

		if len(
			values,
		) > 0 {
			mode =
				strings.ToLower(
					strings.TrimSpace(
						values[len(values)-1],
					),
				)
		}
	}

	switch mode {

	case "",
		"file":

		return "file", nil

	case "none":

		return "none", nil

	default:

		return "",
			fmt.Errorf(
				"unsupported atlas:txmode %q in %s",
				mode,
				file.Name(),
			)
	}
}
