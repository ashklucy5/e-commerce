package productrequest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"project.local/commerce-api/internal/notification"
)

const adminRequestSelect = `
	SELECT
		r.id::text,
		r.request_number,
		r.case_id::text,
		c.case_number,
		r.requested_product_name,
		r.description,
		COALESCE(c.requested_quantity, 0),
		COALESCE(r.customer_requirements::text, ''),
		COALESCE(r.external_url, ''),
		COALESCE(r.attachments::text, ''),
		r.status,
		COALESCE(r.status_reason, ''),
		c.status,
		c.priority,
		customer.id::text,
		customer.full_name,
		customer.phone,
		COALESCE(customer.email, ''),
		COALESCE(reviewer.id::text, ''),
		COALESCE(reviewer.actor_code, ''),
		COALESCE(reviewer.actor_type, ''),
		COALESCE(reviewer.display_name, ''),
		COALESCE(active_assignment.queue_code, ''),
		COALESCE(active_assignment.queue_name, ''),
		COALESCE(active_assignment.actor_id, ''),
		COALESCE(active_assignment.actor_code, ''),
		COALESCE(active_assignment.actor_type, ''),
		COALESCE(active_assignment.actor_name, ''),
		c.last_message_at,
		c.last_customer_message_at,
		c.last_support_message_at,
		r.reviewed_at,
		r.created_at,
		r.updated_at
	FROM product_sourcing_requests r
	JOIN crm_cases c
		ON c.id = r.case_id
	JOIN customers customer
		ON customer.id = c.customer_id
	LEFT JOIN support_actors reviewer
		ON reviewer.id = r.reviewed_by
	LEFT JOIN LATERAL (
		SELECT
			q.code AS queue_code,
			q.name AS queue_name,
			COALESCE(a.id::text, '') AS actor_id,
			COALESCE(a.actor_code, '') AS actor_code,
			COALESCE(a.actor_type, '') AS actor_type,
			COALESCE(a.display_name, '') AS actor_name
		FROM support_case_assignments assignment
		JOIN support_queues q
			ON q.id = assignment.queue_id
		LEFT JOIN support_actors a
			ON a.id = assignment.support_actor_id
		WHERE
			assignment.case_id = c.id
			AND assignment.released_at IS NULL
		ORDER BY assignment.assigned_at DESC
		LIMIT 1
	) active_assignment ON true
`

func scanAdminRequest(
	row rowScanner,
) (
	AdminRequest,
	error,
) {
	var result AdminRequest

	var requirementsText string
	var attachmentsText string

	var reviewerID string
	var reviewerCode string
	var reviewerType string
	var reviewerName string

	var queueCode string
	var queueName string

	var assignmentActorID string
	var assignmentActorCode string
	var assignmentActorType string
	var assignmentActorName string

	var lastCustomerMessageAt pgtype.Timestamptz
	var lastSupportMessageAt pgtype.Timestamptz
	var reviewedAt pgtype.Timestamptz

	err := row.Scan(
		&result.ID,
		&result.RequestNumber,
		&result.CaseID,
		&result.CaseNumber,
		&result.RequestedProductName,
		&result.Description,
		&result.RequestedQuantity,
		&requirementsText,
		&result.ExternalURL,
		&attachmentsText,
		&result.Status,
		&result.StatusReason,
		&result.CRMStatus,
		&result.CRMPriority,
		&result.Customer.ID,
		&result.Customer.FullName,
		&result.Customer.Phone,
		&result.Customer.Email,
		&reviewerID,
		&reviewerCode,
		&reviewerType,
		&reviewerName,
		&queueCode,
		&queueName,
		&assignmentActorID,
		&assignmentActorCode,
		&assignmentActorType,
		&assignmentActorName,
		&result.LastMessageAt,
		&lastCustomerMessageAt,
		&lastSupportMessageAt,
		&reviewedAt,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return AdminRequest{}, err
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

	if reviewerID != "" {
		result.ReviewedBy =
			&AdminActor{
				ID: reviewerID,

				ActorCode: reviewerCode,

				ActorType: reviewerType,

				DisplayName: reviewerName,
			}
	}

	if queueCode != "" {
		assignment :=
			&AdminAssignment{
				QueueCode: queueCode,

				QueueName: queueName,
			}

		if assignmentActorID != "" {
			assignment.SupportActor =
				&AdminActor{
					ID: assignmentActorID,

					ActorCode: assignmentActorCode,

					ActorType: assignmentActorType,

					DisplayName: assignmentActorName,
				}
		}

		result.Assignment =
			assignment
	}

	if lastCustomerMessageAt.Valid {
		value :=
			lastCustomerMessageAt.Time

		result.LastCustomerMessageAt =
			&value
	}

	if lastSupportMessageAt.Valid {
		value :=
			lastSupportMessageAt.Time

		result.LastSupportMessageAt =
			&value
	}

	if reviewedAt.Valid {
		value :=
			reviewedAt.Time

		result.ReviewedAt =
			&value
	}

	return result,
		nil
}

func (r *Repository) ListAdminRequests(
	ctx context.Context,
	filter AdminListFilter,
) (
	[]AdminRequest,
	error,
) {
	rows, err :=
		r.db.Query(
			ctx,
			adminRequestSelect+`
				WHERE
					(
						$1 = ''
						OR r.status = $1
					)
					AND (
						$2 = ''
						OR r.request_number
							ILIKE '%' || $2 || '%'
						OR r.requested_product_name
							ILIKE '%' || $2 || '%'
						OR customer.full_name
							ILIKE '%' || $2 || '%'
						OR customer.phone
							ILIKE '%' || $2 || '%'
						OR COALESCE(
							customer.email,
							''
						) ILIKE '%' || $2 || '%'
					)
				ORDER BY
					r.updated_at DESC,
					r.id DESC
				LIMIT $3
				OFFSET $4
			`,
			filter.Status,
			filter.Query,
			filter.Limit,
			filter.Offset,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list Admin product requests: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]AdminRequest,
			0,
		)

	for rows.Next() {
		item, err :=
			scanAdminRequest(
				rows,
			)
		if err != nil {
			return nil,
				fmt.Errorf(
					"scan Admin product request: %w",
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
				"iterate Admin product requests: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) GetAdminRequest(
	ctx context.Context,
	requestID string,
) (
	AdminRequest,
	error,
) {
	result, err :=
		scanAdminRequest(
			r.db.QueryRow(
				ctx,
				adminRequestSelect+`
					WHERE r.id = $1::uuid
				`,
				requestID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return AdminRequest{},
			ErrNotFound
	}

	if err != nil {
		return AdminRequest{},
			fmt.Errorf(
				"get Admin product request: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) ListAdminMessages(
	ctx context.Context,
	requestID string,
	limit int,
	offset int,
) (
	[]AdminMessage,
	error,
) {
	var caseID string

	err :=
		r.db.QueryRow(
			ctx,
			`
				SELECT
					case_id::text
				FROM product_sourcing_requests
				WHERE id = $1::uuid
			`,
			requestID,
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
				"load Admin product request case: %w",
				err,
			)
	}

	rows, err :=
		r.db.Query(
			ctx,
			`
				SELECT
					m.id::text,
					m.author_type,
					m.visibility,
					m.body,
					COALESCE(
						a.id::text,
						''
					),
					COALESCE(
						a.actor_code,
						''
					),
					COALESCE(
						a.actor_type,
						''
					),
					COALESCE(
						a.display_name,
						''
					),
					m.created_at,
					COALESCE(
						m.attachments::text,
						''
					)
				FROM crm_messages m
				LEFT JOIN support_actors a
					ON a.id =
						m.support_actor_id
				WHERE
					m.case_id =
						$1::uuid
				ORDER BY
					m.created_at ASC,
					m.id ASC
				LIMIT $2
				OFFSET $3
			`,
			caseID,
			limit,
			offset,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"list Admin product request messages: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			[]AdminMessage,
			0,
		)

	for rows.Next() {
		var item AdminMessage

		var actorID string
		var actorCode string
		var actorType string
		var actorName string

		var attachmentsText string

		if err :=
			rows.Scan(
				&item.ID,
				&item.AuthorType,
				&item.Visibility,
				&item.Body,
				&actorID,
				&actorCode,
				&actorType,
				&actorName,
				&item.CreatedAt,
				&attachmentsText,
			); err != nil {

			return nil,
				fmt.Errorf(
					"scan Admin product request message: %w",
					err,
				)
		}

		if actorID != "" {
			item.SupportActor =
				&AdminActor{
					ID: actorID,

					ActorCode: actorCode,

					ActorType: actorType,

					DisplayName: actorName,
				}
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

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"iterate Admin product request messages: %w",
				err,
			)
	}

	return result,
		nil
}

type adminLockedRequest struct {
	CaseID string

	RequestStatus string
	CRMStatus     string
}

func (r *Repository) LockAdminRequestTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
) (
	adminLockedRequest,
	error,
) {
	var result adminLockedRequest

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					r.case_id::text,
					r.status,
					c.status
				FROM product_sourcing_requests r
				JOIN crm_cases c
					ON c.id =
						r.case_id
				WHERE
					r.id =
						$1::uuid
				FOR UPDATE OF r, c
			`,
			requestID,
		).Scan(
			&result.CaseID,
			&result.RequestStatus,
			&result.CRMStatus,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return adminLockedRequest{},
			ErrNotFound
	}

	if err != nil {
		return adminLockedRequest{},
			fmt.Errorf(
				"lock Admin product request: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) EnsureAdminSupportActorTx(
	ctx context.Context,
	tx pgx.Tx,
	staffAccountID string,
) (
	AdminActor,
	error,
) {
	if _, err :=
		tx.Exec(
			ctx,
			`
				SELECT
					pg_advisory_xact_lock(
						hashtext(
							'product-request-admin-actor:'
							|| $1
						)
					)
			`,
			staffAccountID,
		); err != nil {

		return AdminActor{},
			fmt.Errorf(
				"lock Admin support actor: %w",
				err,
			)
	}

	var result AdminActor
	var status string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					id::text,
					actor_code,
					actor_type,
					display_name,
					status
				FROM support_actors
				WHERE
					staff_account_id =
						$1::uuid
				FOR UPDATE
			`,
			staffAccountID,
		).Scan(
			&result.ID,
			&result.ActorCode,
			&result.ActorType,
			&result.DisplayName,
			&status,
		)

	if err == nil {
		if result.ActorType != "human" ||
			status != "active" {

			return AdminActor{},
				ErrAdminActorUnavailable
		}

		return result,
			nil
	}

	if !errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return AdminActor{},
			fmt.Errorf(
				"load Admin support actor: %w",
				err,
			)
	}

	err =
		tx.QueryRow(
			ctx,
			`
				INSERT INTO support_actors (
					actor_code,
					actor_type,
					staff_account_id,
					display_name,
					status,
					presence,
					max_active_cases,
					created_at,
					updated_at
				)
				SELECT
					'SUP-H-' ||
					upper(
						substring(
							replace(
								gen_random_uuid()::text,
								'-',
								''
							)
							FROM 1
							FOR 12
						)
					),
					'human',
					s.id,
					s.full_name,
					'active',
					'offline',
					10,
					now(),
					now()
				FROM staff_accounts s
				WHERE
					s.id =
						$1::uuid
					AND s.status =
						'active'
				RETURNING
					id::text,
					actor_code,
					actor_type,
					display_name
			`,
			staffAccountID,
		).Scan(
			&result.ID,
			&result.ActorCode,
			&result.ActorType,
			&result.DisplayName,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return AdminActor{},
			ErrAdminActorUnavailable
	}

	if err != nil {
		return AdminActor{},
			fmt.Errorf(
				"create Admin support actor: %w",
				err,
			)
	}

	return result,
		nil
}

func (r *Repository) UpdateAdminRequestStatusTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	status string,
	reason string,
	actorID string,
) error {
	if _, err :=
		tx.Exec(
			ctx,
			`
				UPDATE product_sourcing_requests
				SET
					status = $2,
					status_reason =
						NULLIF(
							$3,
							''
						),
					reviewed_by =
						$4::uuid,
					reviewed_at =
						now(),
					updated_at =
						now()
				WHERE
					id =
						$1::uuid
			`,
			requestID,
			status,
			reason,
			actorID,
		); err != nil {

		return fmt.Errorf(
			"update Admin product request status: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) UpdateAdminCRMStatusTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	status string,
) error {
	if _, err := tx.Exec(
		ctx,
		`
			UPDATE crm_cases
			SET
				status = $2::varchar,

				resolved_at =
					CASE
						WHEN $2::varchar = 'closed'
						THEN
							COALESCE(
								resolved_at,
								now()
							)

						WHEN $2::varchar IN (
							'waiting_support',
							'waiting_customer'
						)
						THEN NULL

						ELSE
							resolved_at
					END,

				closed_at =
					CASE
						WHEN $2::varchar = 'closed'
						THEN now()

						ELSE NULL
					END,

				updated_at = now()

			WHERE id = $1::uuid
		`,
		caseID,
		status,
	); err != nil {
		return fmt.Errorf(
			"update product request CRM status: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) InsertAdminEventTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	actorID string,
	eventType string,
	fromCRMStatus string,
	toCRMStatus string,
	payload any,
) error {
	data, err :=
		json.Marshal(
			payload,
		)
	if err != nil {
		return fmt.Errorf(
			"encode Admin product request event: %w",
			err,
		)
	}

	if _, err :=
		tx.Exec(
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
					'support',
					$3::uuid,
					NULLIF(
						$4,
						''
					),
					NULLIF(
						$5,
						''
					),
					$6::jsonb,
					now()
				)
			`,
			caseID,
			eventType,
			actorID,
			fromCRMStatus,
			toCRMStatus,
			string(
				data,
			),
		); err != nil {

		return fmt.Errorf(
			"insert Admin product request event: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) InsertAdminMessageTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	actor AdminActor,
	body string,
	visibility string,
	attachments json.RawMessage,
) (
	AdminMessage,
	error,
) {
	var result AdminMessage

	err :=
		tx.QueryRow(
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
					'support',
					$2::uuid,
					$3,
					$4,
					$5::jsonb,
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
			actor.ID,
			visibility,
			body,
			jsonArgument(
				attachments,
			),
		).Scan(
			&result.ID,
			&result.AuthorType,
			&result.Visibility,
			&result.Body,
			&result.CreatedAt,
		)
	if err != nil {
		return AdminMessage{},
			fmt.Errorf(
				"insert Admin product request message: %w",
				err,
			)
	}

	result.SupportActor =
		&actor

	result.Attachments =
		cloneRawMessage(
			attachments,
		)

	return result,
		nil
}

func (r *Repository) MarkAdminMessageTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	requestID string,
	visibility string,
	currentCRMStatus string,
) (
	string,
	error,
) {
	nextCRMStatus := currentCRMStatus

	if visibility == "customer" {
		nextCRMStatus = "waiting_customer"
	}

	if _, err := tx.Exec(
		ctx,
		`
			UPDATE crm_cases
			SET
				status = $2::varchar,
				last_message_at = now(),
				last_support_message_at = now(),
				resolved_at =
					CASE
						WHEN $2::varchar IN (
							'waiting_support',
							'waiting_customer'
						)
						THEN NULL

						ELSE
							resolved_at
					END,
				updated_at = now()
			WHERE id = $1::uuid
		`,
		caseID,
		nextCRMStatus,
	); err != nil {
		return "",
			fmt.Errorf(
				"update Admin product request message state: %w",
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
		return "",
			fmt.Errorf(
				"update Admin product request activity: %w",
				err,
			)
	}

	return nextCRMStatus,
		nil
}

func (r *Repository) EnqueueAdminReplyNotificationTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	messageID string,
) error {
	var customerID string
	var requestNumber string
	var phone string
	var email string

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					c.customer_id::text,
					r.request_number,
					customer.phone,
					COALESCE(
						customer.email,
						''
					)
				FROM crm_cases c
				JOIN customers customer
					ON customer.id =
						c.customer_id
				JOIN product_sourcing_requests r
					ON r.case_id =
						c.id
				WHERE
					c.id =
						$1::uuid
			`,
			caseID,
		).Scan(
			&customerID,
			&requestNumber,
			&phone,
			&email,
		)
	if err != nil {
		return fmt.Errorf(
			"load product request notification snapshot: %w",
			err,
		)
	}

	message :=
		fmt.Sprintf(
			"You have a new support reply for product request %s.",
			requestNumber,
		)

	if err :=
		notification.EnqueueCustomerEventTx(
			ctx,
			tx,
			notification.CustomerEventRequest{
				BaseDedupeKey: "product-request:" +
					caseID +
					":message:" +
					messageID,

				Category: notification.CategorySourcing,

				EventType: notification.EventSupportReply,

				CustomerID: customerID,

				CaseID: caseID,

				Phone: phone,

				Email: email,

				TemplateKey: notification.TemplateSupportReply,

				Message: message,

				Title:     "Product request update",
				ActionURL: "/account/request",

				Payload: map[string]any{
					"request_number": requestNumber,
				},
			},
		); err != nil {

		return fmt.Errorf(
			"enqueue product request reply notification: %w",
			err,
		)
	}

	return nil
}
