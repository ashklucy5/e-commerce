package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ariga.io/atlas/sql/migrate"
)

type sqlRevisionDB interface {
	ExecContext(
		context.Context,
		string,
		...any,
	) (
		sql.Result,
		error,
	)

	QueryContext(
		context.Context,
		string,
		...any,
	) (
		*sql.Rows,
		error,
	)

	QueryRowContext(
		context.Context,
		string,
		...any,
	) *sql.Row
}

type atlasRevisionStore struct {
	db sqlRevisionDB
}

func (
	s *atlasRevisionStore,
) Ident() *migrate.TableIdent {
	return &migrate.TableIdent{
		Name: atlasRevisionTable,

		Schema: atlasRevisionSchema,
	}
}

func (
	s *atlasRevisionStore,
) ensure(
	ctx context.Context,
) error {
	_, err :=
		s.db.ExecContext(
			ctx,
			`CREATE TABLE IF NOT EXISTS public.atlas_schema_revisions (
				version character varying NOT NULL,
				description character varying NOT NULL,
				type bigint NOT NULL DEFAULT 2,
				applied bigint NOT NULL DEFAULT 0,
				total bigint NOT NULL DEFAULT 0,
				executed_at timestamptz NOT NULL,
				execution_time bigint NOT NULL,
				error text NULL,
				error_stmt text NULL,
				hash character varying NOT NULL,
				partial_hashes jsonb NULL,
				operator_version character varying NOT NULL,
				PRIMARY KEY (version)
			)`,
		)
	if err != nil {
		return fmt.Errorf(
			"ensure Atlas revision table: %w",
			err,
		)
	}

	return nil
}

const atlasRevisionColumns = `
	version,
	description,
	type,
	applied,
	total,
	executed_at,
	execution_time,
	error,
	error_stmt,
	hash,
	partial_hashes,
	operator_version`

func (
	s *atlasRevisionStore,
) ReadRevisions(
	ctx context.Context,
) (
	[]*migrate.Revision,
	error,
) {
	rows, err :=
		s.db.QueryContext(
			ctx,
			`SELECT `+
				atlasRevisionColumns+
				` FROM public.atlas_schema_revisions
				   ORDER BY version`,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"read Atlas revisions: %w",
				err,
			)
	}

	defer rows.Close()

	var revisions []*migrate.Revision

	for rows.Next() {
		revision, err :=
			scanAtlasRevision(
				rows,
			)
		if err != nil {
			return nil, err
		}

		revisions =
			append(
				revisions,
				revision,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate Atlas revisions: %w",
				err,
			)
	}

	return revisions, nil
}

func (
	s *atlasRevisionStore,
) ReadRevision(
	ctx context.Context,
	version string,
) (
	*migrate.Revision,
	error,
) {
	revision, err :=
		scanAtlasRevision(
			s.db.QueryRowContext(
				ctx,
				`SELECT `+
					atlasRevisionColumns+
					` FROM public.atlas_schema_revisions
					   WHERE version = $1`,
				version,
			),
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			migrate.ErrRevisionNotExist
	}

	if err != nil {
		return nil, err
	}

	return revision, nil
}

func (
	s *atlasRevisionStore,
) WriteRevision(
	ctx context.Context,
	revision *migrate.Revision,
) error {
	partialHashes :=
		""

	if len(
		revision.PartialHashes,
	) > 0 {
		data, err :=
			json.Marshal(
				revision.PartialHashes,
			)
		if err != nil {
			return fmt.Errorf(
				"encode Atlas partial hashes: %w",
				err,
			)
		}

		partialHashes =
			string(
				data,
			)
	}

	_, err :=
		s.db.ExecContext(
			ctx,
			`INSERT INTO public.atlas_schema_revisions (
				version,
				description,
				type,
				applied,
				total,
				executed_at,
				execution_time,
				error,
				error_stmt,
				hash,
				partial_hashes,
				operator_version
			) VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
				$6,
				$7,
				NULLIF($8, ''),
				NULLIF($9, ''),
				$10,
				NULLIF($11, '')::jsonb,
				$12
			)
			ON CONFLICT (version) DO UPDATE SET
				description = EXCLUDED.description,
				type = EXCLUDED.type,
				applied = EXCLUDED.applied,
				total = EXCLUDED.total,
				executed_at = EXCLUDED.executed_at,
				execution_time = EXCLUDED.execution_time,
				error = EXCLUDED.error,
				error_stmt = EXCLUDED.error_stmt,
				hash = EXCLUDED.hash,
				partial_hashes = EXCLUDED.partial_hashes,
				operator_version = EXCLUDED.operator_version`,
			revision.Version,
			revision.Description,
			int64(
				revision.Type,
			),
			revision.Applied,
			revision.Total,
			revision.ExecutedAt,
			int64(
				revision.ExecutionTime,
			),
			revision.Error,
			revision.ErrorStmt,
			revision.Hash,
			partialHashes,
			revision.OperatorVersion,
		)
	if err != nil {
		return fmt.Errorf(
			"write Atlas revision %s: %w",
			revision.Version,
			err,
		)
	}

	return nil
}

func (
	s *atlasRevisionStore,
) DeleteRevision(
	ctx context.Context,
	version string,
) error {
	_, err :=
		s.db.ExecContext(
			ctx,
			`DELETE
			 FROM public.atlas_schema_revisions
			 WHERE version = $1`,
			version,
		)
	if err != nil {
		return fmt.Errorf(
			"delete Atlas revision %s: %w",
			version,
			err,
		)
	}

	return nil
}

type revisionScanner interface {
	Scan(
		...any,
	) error
}

func scanAtlasRevision(
	scanner revisionScanner,
) (
	*migrate.Revision,
	error,
) {
	var (
		revision migrate.Revision

		revisionType int64

		applied int64

		total int64

		executionTime int64

		errorText sql.NullString

		errorStmt sql.NullString

		partialHashesJSON []byte
	)

	if err :=
		scanner.Scan(
			&revision.Version,
			&revision.Description,
			&revisionType,
			&applied,
			&total,
			&revision.ExecutedAt,
			&executionTime,
			&errorText,
			&errorStmt,
			&revision.Hash,
			&partialHashesJSON,
			&revision.OperatorVersion,
		); err != nil {

		return nil, err
	}

	revision.Type =
		migrate.RevisionType(
			revisionType,
		)

	revision.Applied =
		int(
			applied,
		)

	revision.Total =
		int(
			total,
		)

	revision.ExecutionTime =
		time.Duration(
			executionTime,
		)

	if errorText.Valid {
		revision.Error =
			errorText.String
	}

	if errorStmt.Valid {
		revision.ErrorStmt =
			errorStmt.String
	}

	if len(
		partialHashesJSON,
	) > 0 {
		if err :=
			json.Unmarshal(
				partialHashesJSON,
				&revision.PartialHashes,
			); err != nil {

			return nil,
				fmt.Errorf(
					"decode Atlas partial hashes for %s: %w",
					revision.Version,
					err,
				)
		}
	}

	return &revision, nil
}
