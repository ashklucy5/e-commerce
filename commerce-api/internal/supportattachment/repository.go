package supportattachment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"begin support attachment transaction: %w",
			err,
		)
	}

	return tx, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

const attachmentColumns = `
	id::text,
	COALESCE(case_id::text, ''),
	COALESCE(message_id::text, ''),
	uploader_type,
	COALESCE(customer_id::text, ''),
	COALESCE(support_actor_id::text, ''),
	storage_key,
	original_filename,
	mime_type,
	byte_size,
	status,
	uploaded_at,
	deleted_at,
	created_at,
	updated_at
`

func scanAttachment(row rowScanner) (Attachment, error) {
	var result Attachment

	var uploadedAt pgtype.Timestamptz
	var deletedAt pgtype.Timestamptz
	var uploaderType string
	var status string

	err := row.Scan(
		&result.ID,
		&result.CaseID,
		&result.MessageID,
		&uploaderType,
		&result.CustomerID,
		&result.SupportActorID,
		&result.StorageKey,
		&result.OriginalFilename,
		&result.MimeType,
		&result.ByteSize,
		&status,
		&uploadedAt,
		&deletedAt,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return Attachment{}, err
	}

	result.UploaderType = UploaderType(uploaderType)
	result.Status = Status(status)

	if uploadedAt.Valid {
		value := uploadedAt.Time.UTC()
		result.UploadedAt = &value
	}

	if deletedAt.Valid {
		value := deletedAt.Time.UTC()
		result.DeletedAt = &value
	}

	return result, nil
}

func (r *Repository) CreatePending(
	ctx context.Context,
	input CreatePendingInput,
) (Attachment, error) {
	query := `
		INSERT INTO crm_support_attachments (
			id,
			case_id,
			message_id,
			uploader_type,
			customer_id,
			support_actor_id,
			storage_key,
			original_filename,
			mime_type,
			byte_size,
			status,
			created_at,
			updated_at
		)
		VALUES (
			$1::uuid,
			NULLIF($2, '')::uuid,
			NULL,
			$3,
			NULLIF($4, '')::uuid,
			NULLIF($5, '')::uuid,
			$6,
			$7,
			$8,
			$9,
			'pending',
			now(),
			now()
		)
		RETURNING
	` + attachmentColumns

	result, err := scanAttachment(
		r.db.QueryRow(
			ctx,
			query,
			input.ID,
			input.CaseID,
			string(input.UploaderType),
			input.CustomerID,
			input.SupportActorID,
			input.StorageKey,
			input.OriginalFilename,
			input.MimeType,
			input.ByteSize,
		),
	)
	if err != nil {
		return Attachment{}, fmt.Errorf(
			"create pending support attachment: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	attachmentID string,
) (Attachment, error) {
	query := `
		SELECT
	` + attachmentColumns + `
		FROM crm_support_attachments
		WHERE id = $1::uuid
		LIMIT 1
	`

	result, err := scanAttachment(
		r.db.QueryRow(
			ctx,
			query,
			attachmentID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}

	if err != nil {
		return Attachment{}, fmt.Errorf(
			"load support attachment: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) MarkReady(
	ctx context.Context,
	attachmentID string,
	uploaderType UploaderType,
	ownerID string,
) (Attachment, error) {
	query := `
		UPDATE crm_support_attachments
		SET
			status = 'ready',
			uploaded_at = now(),
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'pending'
			AND uploader_type = $2
			AND (
				(
					$2 = 'customer'
					AND customer_id = $3::uuid
				)
				OR
				(
					$2 = 'support'
					AND support_actor_id = $3::uuid
				)
			)
		RETURNING
	` + attachmentColumns

	result, err := scanAttachment(
		r.db.QueryRow(
			ctx,
			query,
			attachmentID,
			string(uploaderType),
			ownerID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrInvalidState
	}

	if err != nil {
		return Attachment{}, fmt.Errorf(
			"mark support attachment ready: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) LockReadyOwnedTx(
	ctx context.Context,
	tx pgx.Tx,
	attachmentID string,
	uploaderType UploaderType,
	ownerID string,
	caseID string,
) (Attachment, error) {
	query := `
		SELECT
	` + attachmentColumns + `
		FROM crm_support_attachments
		WHERE
			id = $1::uuid
			AND status = 'ready'
			AND uploader_type = $2
			AND (
				(
					$2 = 'customer'
					AND customer_id = $3::uuid
				)
				OR
				(
					$2 = 'support'
					AND support_actor_id = $3::uuid
				)
			)
			AND (
				case_id IS NULL
				OR case_id = $4::uuid
			)
		FOR UPDATE
	`

	result, err := scanAttachment(
		tx.QueryRow(
			ctx,
			query,
			attachmentID,
			string(uploaderType),
			ownerID,
			caseID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrInvalidState
	}

	if err != nil {
		return Attachment{}, fmt.Errorf(
			"lock ready support attachment: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) BindReadyToMessageTx(
	ctx context.Context,
	tx pgx.Tx,
	attachmentID string,
	uploaderType UploaderType,
	ownerID string,
	caseID string,
	messageID string,
) (Attachment, error) {
	query := `
		UPDATE crm_support_attachments
		SET
			case_id = $4::uuid,
			message_id = $5::uuid,
			status = 'attached',
			updated_at = now()
		WHERE
			id = $1::uuid
			AND status = 'ready'
			AND uploader_type = $2
			AND (
				(
					$2 = 'customer'
					AND customer_id = $3::uuid
				)
				OR
				(
					$2 = 'support'
					AND support_actor_id = $3::uuid
				)
			)
			AND (
				case_id IS NULL
				OR case_id = $4::uuid
			)
		RETURNING
	` + attachmentColumns

	result, err := scanAttachment(
		tx.QueryRow(
			ctx,
			query,
			attachmentID,
			string(uploaderType),
			ownerID,
			caseID,
			messageID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrInvalidState
	}

	if err != nil {
		return Attachment{}, fmt.Errorf(
			"bind support attachment to message: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) ListRetentionCandidates(
	ctx context.Context,
	closedBefore time.Time,
	limit int,
) ([]RetentionCandidate, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.Query(
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
		return nil, fmt.Errorf(
			"list support attachment retention candidates: %w",
			err,
		)
	}

	defer rows.Close()

	result := make([]RetentionCandidate, 0)

	for rows.Next() {
		var item RetentionCandidate

		if err := rows.Scan(
			&item.ID,
			&item.StorageKey,
		); err != nil {
			return nil, fmt.Errorf(
				"scan support attachment retention candidate: %w",
				err,
			)
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate support attachment retention candidates: %w",
			err,
		)
	}

	return result, nil
}

func (r *Repository) MarkDeleted(
	ctx context.Context,
	attachmentID string,
) error {
	commandTag, err := r.db.Exec(
		ctx,
		`
			UPDATE crm_support_attachments
			SET
				status = 'deleted',
				deleted_at = now(),
				updated_at = now()
			WHERE
				id = $1::uuid
				AND status = 'attached'
				AND deleted_at IS NULL
		`,
		attachmentID,
	)
	if err != nil {
		return fmt.Errorf(
			"mark support attachment deleted: %w",
			err,
		)
	}

	if commandTag.RowsAffected() == 0 {
		return ErrInvalidState
	}

	return nil
}
