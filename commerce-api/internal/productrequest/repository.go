package productrequest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"begin product request transaction: %w",
			err,
		)
	}

	return tx, nil
}

type createCaseInput struct {
	CaseNumber        string
	CustomerID        string
	Subject           string
	RequestedQuantity int
	ContextSnapshot   json.RawMessage
}

func (r *Repository) CreateCaseTx(
	ctx context.Context,
	tx pgx.Tx,
	input createCaseInput,
) (string, error) {
	var id string

	err := tx.QueryRow(
		ctx,
		`
			INSERT INTO crm_cases (
				case_number,
				customer_id,
				case_type,
				subject,
				status,
				priority,
				requested_quantity,
				context_snapshot,
				last_message_at,
				last_customer_message_at,
				created_at,
				updated_at
			)
			VALUES (
				$1,
				$2::uuid,
				'product_request',
				$3,
				'waiting_support',
				'normal',
				$4,
				$5::jsonb,
				now(),
				now(),
				now(),
				now()
			)
			RETURNING id::text
		`,
		input.CaseNumber,
		input.CustomerID,
		input.Subject,
		input.RequestedQuantity,
		string(input.ContextSnapshot),
	).Scan(
		&id,
	)
	if err != nil {
		return "", fmt.Errorf(
			"create product request CRM case: %w",
			err,
		)
	}

	return id, nil
}

type createSourcingRequestInput struct {
	CaseID               string
	RequestNumber        string
	RequestedProductName string
	Description          string
	CustomerRequirements json.RawMessage
	ExternalURL          string
	Attachments          json.RawMessage
}

func (r *Repository) CreateSourcingRequestTx(
	ctx context.Context,
	tx pgx.Tx,
	input createSourcingRequestInput,
) (string, error) {
	var id string

	err := tx.QueryRow(
		ctx,
		`
			INSERT INTO product_sourcing_requests (
				case_id,
				request_number,
				requested_product_name,
				description,
				customer_requirements,
				external_url,
				attachments,
				status,
				status_reason,
				created_at,
				updated_at
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				$4,
				$5::jsonb,
				NULLIF($6, ''),
				$7::jsonb,
				'pending_review',
				NULL,
				now(),
				now()
			)
			RETURNING id::text
		`,
		input.CaseID,
		input.RequestNumber,
		input.RequestedProductName,
		input.Description,
		jsonArgument(input.CustomerRequirements),
		input.ExternalURL,
		jsonArgument(input.Attachments),
	).Scan(
		&id,
	)
	if err != nil {
		return "", fmt.Errorf(
			"create product sourcing request: %w",
			err,
		)
	}

	return id, nil
}

func (r *Repository) InsertCustomerMessageTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	body string,
	attachments json.RawMessage,
) (Message, error) {
	var result Message

	err := tx.QueryRow(
		ctx,
		`
			INSERT INTO crm_messages (
				case_id,
				author_type,
				support_actor_id,
				visibility,
				body,
				attachments,
				created_at
			)
			VALUES (
				$1::uuid,
				'customer',
				NULL,
				'customer',
				$2,
				$3::jsonb,
				now()
			)
			RETURNING
				id::text,
				author_type,
				visibility,
				body,
				created_at
		`,
		caseID,
		body,
		jsonArgument(attachments),
	).Scan(
		&result.ID,
		&result.AuthorType,
		&result.Visibility,
		&result.Body,
		&result.CreatedAt,
	)
	if err != nil {
		return Message{}, fmt.Errorf(
			"insert product request customer message: %w",
			err,
		)
	}

	result.Attachments = cloneRawMessage(attachments)

	return result, nil
}

func (r *Repository) InsertCustomerEventTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	eventType string,
	fromStatus string,
	toStatus string,
	payload any,
) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf(
			"encode product request CRM event payload: %w",
			err,
		)
	}

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO crm_case_events (
				case_id,
				event_type,
				actor_type,
				support_actor_id,
				from_status,
				to_status,
				payload,
				created_at
			)
			VALUES (
				$1::uuid,
				$2,
				'customer',
				NULL,
				NULLIF($3, ''),
				NULLIF($4, ''),
				$5::jsonb,
				now()
			)
		`,
		caseID,
		eventType,
		fromStatus,
		toStatus,
		string(data),
	)
	if err != nil {
		return fmt.Errorf(
			"insert product request CRM event: %w",
			err,
		)
	}

	return nil
}

const requestSelect = `
	SELECT
		r.id::text,
		r.request_number,
		r.requested_product_name,
		r.description,
		COALESCE(c.requested_quantity, 0),
		COALESCE(r.customer_requirements::text, ''),
		COALESCE(r.external_url, ''),
		COALESCE(r.attachments::text, ''),
		r.status,
		COALESCE(r.status_reason, ''),
		c.status,
		c.last_message_at,
		r.reviewed_at,
		r.created_at,
		r.updated_at
	FROM product_sourcing_requests r
	JOIN crm_cases c
		ON c.id = r.case_id
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRequest(row rowScanner) (Request, error) {
	var result Request
	var requirementsText string
	var attachmentsText string
	var reviewedAt pgtype.Timestamptz
	var conversationStatus string

	err := row.Scan(
		&result.ID,
		&result.RequestNumber,
		&result.RequestedProductName,
		&result.Description,
		&result.RequestedQuantity,
		&requirementsText,
		&result.ExternalURL,
		&attachmentsText,
		&result.Status,
		&result.StatusReason,
		&conversationStatus,
		&result.LastMessageAt,
		&reviewedAt,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return Request{}, err
	}

	if requirementsText != "" {
		result.CustomerRequirements =
			json.RawMessage(
				requirementsText,
			)
	}

	if attachmentsText != "" {
		result.Attachments =
			json.RawMessage(
				attachmentsText,
			)
	}

	if reviewedAt.Valid {
		value := reviewedAt.Time
		result.ReviewedAt = &value
	}

	result.CanMessage =
		conversationStatus != "closed" &&
			result.Status != "cancelled" &&
			result.Status != "converted_to_order"

	return result, nil
}

func (r *Repository) GetCustomerRequest(
	ctx context.Context,
	customerID string,
	requestID string,
) (Request, error) {
	result, err := scanRequest(
		r.db.QueryRow(
			ctx,
			requestSelect+`
				WHERE
					r.id = $1::uuid
					AND c.customer_id = $2::uuid
			`,
			requestID,
			customerID,
		),
	)
	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return Request{},
			ErrNotFound
	}

	if err != nil {
		return Request{},
			fmt.Errorf(
				"get customer product request: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ListCustomerRequests(
	ctx context.Context,
	customerID string,
	limit int,
	offset int,
) ([]Request, error) {
	rows, err := r.db.Query(
		ctx,
		requestSelect+`
			WHERE c.customer_id = $1::uuid
			ORDER BY r.updated_at DESC, r.id DESC
			LIMIT $2 OFFSET $3
		`,
		customerID,
		limit,
		offset,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list customer product requests: %w",
				err,
			)
	}
	defer rows.Close()

	result := make(
		[]Request,
		0,
	)

	for rows.Next() {
		item, err :=
			scanRequest(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan customer product request: %w",
					err,
				)
		}

		result =
			append(
				result,
				item,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate customer product requests: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) ListCustomerMessages(
	ctx context.Context,
	customerID string,
	requestID string,
	limit int,
	offset int,
) ([]Message, error) {
	var caseID string

	err := r.db.QueryRow(
		ctx,
		`
			SELECT r.case_id::text
			FROM product_sourcing_requests r
			JOIN crm_cases c
				ON c.id = r.case_id
			WHERE
				r.id = $1::uuid
				AND c.customer_id = $2::uuid
		`,
		requestID,
		customerID,
	).Scan(
		&caseID,
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return nil,
			ErrNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"verify product request ownership: %w",
				err,
			)
	}

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				m.id::text,
				m.author_type,
				m.visibility,
				m.body,
				COALESCE(a.actor_code, ''),
				COALESCE(a.actor_type, ''),
				COALESCE(a.display_name, ''),
				m.created_at,
				COALESCE(m.attachments::text, '')
			FROM crm_messages m
			LEFT JOIN support_actors a
				ON a.id = m.support_actor_id
			WHERE
				m.case_id = $1::uuid
				AND m.visibility = 'customer'
			ORDER BY m.created_at ASC, m.id ASC
			LIMIT $2 OFFSET $3
		`,
		caseID,
		limit,
		offset,
	)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list product request messages: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]Message,
			0,
		)

	for rows.Next() {
		var item Message

		var attachmentsText string

		if err := rows.Scan(
			&item.ID,
			&item.AuthorType,
			&item.Visibility,
			&item.Body,
			&item.SupportActorCode,
			&item.SupportActorType,
			&item.SupportActorName,
			&item.CreatedAt,
			&attachmentsText,
		); err != nil {
			return nil,
				fmt.Errorf(
					"scan product request message: %w",
					err,
				)
		}

		if attachmentsText != "" {
			item.Attachments =
				json.RawMessage(
					attachmentsText,
				)
		}

		result =
			append(
				result,
				item,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"iterate product request messages: %w",
				err,
			)
	}

	return result, nil
}

type conversationState struct {
	CaseID             string
	ConversationStatus string
	RequestStatus      string
}

func (r *Repository) LockCustomerRequestTx(
	ctx context.Context,
	tx pgx.Tx,
	customerID string,
	requestID string,
) (conversationState, error) {
	var result conversationState

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				r.case_id::text,
				c.status,
				r.status
			FROM product_sourcing_requests r
			JOIN crm_cases c
				ON c.id = r.case_id
			WHERE
				r.id = $1::uuid
				AND c.customer_id = $2::uuid
			FOR UPDATE OF r, c
		`,
		requestID,
		customerID,
	).Scan(
		&result.CaseID,
		&result.ConversationStatus,
		&result.RequestStatus,
	)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return conversationState{},
			ErrNotFound
	}

	if err != nil {
		return conversationState{},
			fmt.Errorf(
				"lock customer product request: %w",
				err,
			)
	}

	return result, nil
}

func (r *Repository) MarkCustomerMessageTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	requestID string,
	clearResolved bool,
) error {
	if _, err := tx.Exec(
		ctx,
		`
			UPDATE crm_cases
			SET
				status = 'waiting_support',
				last_message_at = now(),
				last_customer_message_at = now(),
				resolved_at = CASE
					WHEN $2 THEN NULL
					ELSE resolved_at
				END,
				updated_at = now()
			WHERE id = $1::uuid
		`,
		caseID,
		clearResolved,
	); err != nil {
		return fmt.Errorf(
			"update product request CRM message state: %w",
			err,
		)
	}

	if _, err := tx.Exec(
		ctx,
		`
			UPDATE product_sourcing_requests
			SET updated_at = now()
			WHERE id = $1::uuid
		`,
		requestID,
	); err != nil {
		return fmt.Errorf(
			"update product request activity time: %w",
			err,
		)
	}

	return nil
}

func jsonArgument(
	value json.RawMessage,
) any {
	if len(value) == 0 {
		return nil
	}

	return string(
		value,
	)
}

func cloneRawMessage(
	value json.RawMessage,
) json.RawMessage {
	if len(value) == 0 {
		return nil
	}

	result :=
		make(
			json.RawMessage,
			len(value),
		)

	copy(
		result,
		value,
	)

	return result
}
