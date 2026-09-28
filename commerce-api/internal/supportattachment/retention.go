package supportattachment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/platform/storage"
)

const (
	RetentionAfterCaseClose = 7 * 24 * time.Hour

	OrphanRetentionAge = 24 * time.Hour

	DefaultRetentionBatchSize = 100

	MaxRetentionBatchSize = 500
)

type RetentionResult struct {
	ClosedCandidates int `json:"closed_candidates"`
	ClosedDeleted    int `json:"closed_deleted"`

	OrphanCandidates int `json:"orphan_candidates"`
	OrphanDeleted    int `json:"orphan_deleted"`

	Failed int `json:"failed"`
}

type retentionCandidate struct {
	ID string

	StorageKey string
}

type RetentionService struct {
	repository *Repository
	storage    *storage.Gateway
}

func NewRetentionService(
	repository *Repository,
	storageGateway *storage.Gateway,
) *RetentionService {
	return &RetentionService{
		repository: repository,
		storage:    storageGateway,
	}
}

func (s *RetentionService) Run(
	ctx context.Context,
	now time.Time,
	limit int,
) (RetentionResult, error) {
	var result RetentionResult

	if s == nil ||
		s.repository == nil ||
		s.storage == nil {

		return result,
			fmt.Errorf(
				"support attachment retention service is not configured",
			)
	}

	now =
		now.UTC()

	limit =
		normalizeRetentionBatchLimit(
			limit,
		)

	closedBefore,
		orphanBefore :=
		retentionCutoffs(
			now,
		)

	closedCandidates, err :=
		s.repository.
			listClosedRetentionCandidates(
				ctx,
				closedBefore,
				limit,
			)
	if err != nil {
		return result, err
	}

	result.ClosedCandidates =
		len(
			closedCandidates,
		)

	orphanCandidates, err :=
		s.repository.
			listOrphanRetentionCandidates(
				ctx,
				orphanBefore,
				limit,
			)
	if err != nil {
		return result, err
	}

	result.OrphanCandidates =
		len(
			orphanCandidates,
		)

	var failures []error

	for _, candidate := range closedCandidates {

		deleted, err :=
			s.deleteClosedCandidate(
				ctx,
				candidate.ID,
				closedBefore,
			)
		if err != nil {
			result.Failed++

			failures =
				append(
					failures,
					fmt.Errorf(
						"delete closed support attachment %s: %w",
						candidate.ID,
						err,
					),
				)

			continue
		}

		if deleted {
			result.ClosedDeleted++
		}
	}

	for _, candidate := range orphanCandidates {

		deleted, err :=
			s.deleteOrphanCandidate(
				ctx,
				candidate.ID,
				orphanBefore,
			)
		if err != nil {
			result.Failed++

			failures =
				append(
					failures,
					fmt.Errorf(
						"delete orphan support attachment %s: %w",
						candidate.ID,
						err,
					),
				)

			continue
		}

		if deleted {
			result.OrphanDeleted++
		}
	}

	return result,
		errors.Join(
			failures...,
		)
}

func (s *RetentionService) deleteClosedCandidate(
	ctx context.Context,
	attachmentID string,
	closedBefore time.Time,
) (bool, error) {
	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return false, err
	}

	defer func() {
		_ =
			tx.Rollback(
				ctx,
			)
	}()

	/*
		Re-check the complete retention condition while holding
		a row lock.

		The initial candidate listing is deliberately not trusted
		as authorization to delete because case state may have
		changed between listing and processing.
	*/
	candidate, found, err :=
		s.repository.
			lockClosedRetentionCandidateTx(
				ctx,
				tx,
				attachmentID,
				closedBefore,
			)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	if err :=
		deleteRetentionObject(
			ctx,
			s.storage,
			candidate.StorageKey,
		); err != nil {

		return false, err
	}

	if err :=
		s.repository.
			markDeletedTx(
				ctx,
				tx,
				candidate.ID,
			); err != nil {

		return false, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return false,
			fmt.Errorf(
				"commit closed support attachment retention: %w",
				err,
			)
	}

	return true, nil
}

func (s *RetentionService) deleteOrphanCandidate(
	ctx context.Context,
	attachmentID string,
	orphanBefore time.Time,
) (bool, error) {
	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return false, err
	}

	defer func() {
		_ =
			tx.Rollback(
				ctx,
			)
	}()

	/*
		This lock is particularly important for ready uploads.

		BindReadyToMessageTx also locks the attachment row. As a
		result, cleanup and message binding cannot race: whichever
		transaction gets the row first determines whether the
		attachment becomes attached or deleted.
	*/
	candidate, found, err :=
		s.repository.
			lockOrphanRetentionCandidateTx(
				ctx,
				tx,
				attachmentID,
				orphanBefore,
			)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	if err :=
		deleteRetentionObject(
			ctx,
			s.storage,
			candidate.StorageKey,
		); err != nil {

		return false, err
	}

	if err :=
		s.repository.
			markDeletedTx(
				ctx,
				tx,
				candidate.ID,
			); err != nil {

		return false, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return false,
			fmt.Errorf(
				"commit orphan support attachment retention: %w",
				err,
			)
	}

	return true, nil
}

func deleteRetentionObject(
	ctx context.Context,
	storageGateway *storage.Gateway,
	storageKey string,
) error {
	err :=
		storageGateway.Delete(
			ctx,
			storageKey,
		)

	/*
		S3/B2 DELETE may report the object as missing.

		For retention purposes this is already the desired object
		state, so the metadata can safely become a deleted
		tombstone.
	*/
	if errors.Is(
		err,
		storage.ErrNotFound,
	) {
		return nil
	}

	if err != nil {
		return fmt.Errorf(
			"delete support attachment object: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) listClosedRetentionCandidates(
	ctx context.Context,
	closedBefore time.Time,
	limit int,
) ([]retentionCandidate, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					a.id::text,
					a.storage_key
				FROM crm_support_attachments a
				JOIN crm_cases c
					ON c.id = a.case_id
				WHERE
					a.status = 'attached'
					AND a.message_id IS NOT NULL
					AND a.case_id IS NOT NULL
					AND a.deleted_at IS NULL
					AND c.status = 'closed'
					AND c.closed_at IS NOT NULL
					AND c.closed_at <= $1
				ORDER BY
					c.closed_at ASC,
					a.created_at ASC,
					a.id ASC
				LIMIT $2
			`,
			closedBefore,
			limit,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list closed support attachment retention candidates: %w",
				err,
			)
	}

	defer rows.Close()

	result :=
		make(
			[]retentionCandidate,
			0,
			limit,
		)

	for rows.Next() {
		var item retentionCandidate

		if err :=
			rows.Scan(
				&item.ID,
				&item.StorageKey,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan closed support attachment retention candidate: %w",
					err,
				)
		}

		result =
			append(
				result,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate closed support attachment retention candidates: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) listOrphanRetentionCandidates(
	ctx context.Context,
	orphanBefore time.Time,
	limit int,
) ([]retentionCandidate, error) {
	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					id::text,
					storage_key
				FROM crm_support_attachments
				WHERE
					message_id IS NULL
					AND deleted_at IS NULL
					AND (
						(
							status = 'pending'
							AND created_at <= $1
						)
						OR
						(
							status = 'ready'
							AND COALESCE(
								uploaded_at,
								created_at
							) <= $1
						)
					)
				ORDER BY
					CASE
						WHEN status = 'pending'
							THEN created_at
						ELSE COALESCE(
							uploaded_at,
							created_at
						)
					END ASC,
					id ASC
				LIMIT $2
			`,
			orphanBefore,
			limit,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list orphan support attachment retention candidates: %w",
				err,
			)
	}

	defer rows.Close()

	result :=
		make(
			[]retentionCandidate,
			0,
			limit,
		)

	for rows.Next() {
		var item retentionCandidate

		if err :=
			rows.Scan(
				&item.ID,
				&item.StorageKey,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan orphan support attachment retention candidate: %w",
					err,
				)
		}

		result =
			append(
				result,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate orphan support attachment retention candidates: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) lockClosedRetentionCandidateTx(
	ctx context.Context,
	tx pgx.Tx,
	attachmentID string,
	closedBefore time.Time,
) (
	retentionCandidate,
	bool,
	error,
) {
	var result retentionCandidate

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					a.id::text,
					a.storage_key
				FROM crm_support_attachments a
				JOIN crm_cases c
					ON c.id = a.case_id
				WHERE
					a.id = $1::uuid
					AND a.status = 'attached'
					AND a.message_id IS NOT NULL
					AND a.case_id IS NOT NULL
					AND a.deleted_at IS NULL
					AND c.status = 'closed'
					AND c.closed_at IS NOT NULL
					AND c.closed_at <= $2
				FOR UPDATE OF a
			`,
			attachmentID,
			closedBefore,
		).Scan(
			&result.ID,
			&result.StorageKey,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return retentionCandidate{},
			false,
			nil
	}

	if err != nil {
		return retentionCandidate{},
			false,
			fmt.Errorf(
				"lock closed support attachment retention candidate: %w",
				err,
			)
	}

	return result,
		true,
		nil
}

func (r *Repository) lockOrphanRetentionCandidateTx(
	ctx context.Context,
	tx pgx.Tx,
	attachmentID string,
	orphanBefore time.Time,
) (
	retentionCandidate,
	bool,
	error,
) {
	var result retentionCandidate

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					storage_key
				FROM crm_support_attachments
				WHERE
					id = $1::uuid
					AND message_id IS NULL
					AND deleted_at IS NULL
					AND (
						(
							status = 'pending'
							AND created_at <= $2
						)
						OR
						(
							status = 'ready'
							AND COALESCE(
								uploaded_at,
								created_at
							) <= $2
						)
					)
				FOR UPDATE
			`,
			attachmentID,
			orphanBefore,
		).Scan(
			&result.ID,
			&result.StorageKey,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return retentionCandidate{},
			false,
			nil
	}

	if err != nil {
		return retentionCandidate{},
			false,
			fmt.Errorf(
				"lock orphan support attachment retention candidate: %w",
				err,
			)
	}

	return result,
		true,
		nil
}

func (r *Repository) markDeletedTx(
	ctx context.Context,
	tx pgx.Tx,
	attachmentID string,
) error {
	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE crm_support_attachments
				SET
					status = 'deleted',
					deleted_at = COALESCE(
						deleted_at,
						now()
					),
					updated_at = now()
				WHERE
					id = $1::uuid
					AND status <> 'deleted'
			`,
			attachmentID,
		)
	if err != nil {
		return fmt.Errorf(
			"mark support attachment deleted: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrInvalidState
	}

	return nil
}

func normalizeRetentionBatchLimit(
	limit int,
) int {
	if limit <= 0 {
		return DefaultRetentionBatchSize
	}

	if limit >
		MaxRetentionBatchSize {

		return MaxRetentionBatchSize
	}

	return limit
}

func retentionCutoffs(
	now time.Time,
) (
	time.Time,
	time.Time,
) {
	now =
		now.UTC()

	return now.Add(
			-RetentionAfterCaseClose,
		),
		now.Add(
			-OrphanRetentionAge,
		)
}
