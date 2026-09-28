package support

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type SupportMessage struct {
	ID     string `json:"id"`
	CaseID string `json:"case_id"`

	AuthorType string `json:"author_type"`
	Visibility string `json:"visibility"`
	Body       string `json:"body"`

	SupportActor *TicketActor `json:"support_actor,omitempty"`

	Attachments json.RawMessage `json:"attachments,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

func (r *PostgresRepository) ListCaseMessagesForActor(
	ctx context.Context,
	actorID string,
	caseID string,
	limit int,
	offset int,
) ([]SupportMessage, error) {
	var allowed bool

	err := r.db.QueryRow(
		ctx,
		`
			SELECT EXISTS (
				SELECT 1
				FROM support_case_assignments a
				JOIN support_queue_members qm
					ON qm.queue_id = a.queue_id
				WHERE
					a.case_id = $1::uuid
					AND a.released_at IS NULL
					AND qm.support_actor_id = $2::uuid
			)
		`,
		caseID,
		actorID,
	).Scan(
		&allowed,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"verify support case access: %w",
			err,
		)
	}

	if !allowed {
		return nil, ErrCaseNotFound
	}

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				m.id::text,
				m.case_id::text,
				m.author_type,
				m.visibility,
				m.body,
				COALESCE(
					m.attachments::text,
					''
				),
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
				m.created_at
			FROM crm_messages m
			LEFT JOIN support_actors a
				ON a.id = m.support_actor_id
			WHERE
				m.case_id = $1::uuid
			ORDER BY
				m.created_at ASC,
				m.id ASC
			LIMIT $2 OFFSET $3
		`,
		caseID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list support case messages: %w",
			err,
		)
	}

	defer rows.Close()

	result := make(
		[]SupportMessage,
		0,
	)

	for rows.Next() {
		var item SupportMessage

		var attachmentsText string
		var supportActorID string
		var actorCode string
		var actorType string
		var actorName string

		if err := rows.Scan(
			&item.ID,
			&item.CaseID,
			&item.AuthorType,
			&item.Visibility,
			&item.Body,
			&attachmentsText,
			&supportActorID,
			&actorCode,
			&actorType,
			&actorName,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"scan support message: %w",
				err,
			)
		}

		if attachmentsText != "" &&
			attachmentsText != "null" {

			item.Attachments = json.RawMessage(
				attachmentsText,
			)
		}

		if supportActorID != "" {
			item.SupportActor = &TicketActor{
				ID:          supportActorID,
				ActorCode:   actorCode,
				ActorType:   actorType,
				DisplayName: actorName,
			}
		}

		result = append(
			result,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate support messages: %w",
			err,
		)
	}

	return result, nil
}

func (r *PostgresRepository) AddSupportMessage(
	ctx context.Context,
	actorID string,
	caseID string,
	body string,
	visibility string,
) (SupportMessage, error) {
	tx, err := r.db.Begin(
		ctx,
	)
	if err != nil {
		return SupportMessage{},
			fmt.Errorf(
				"begin support message transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	assignment, err :=
		r.lockActiveAssignmentTx(
			ctx,
			tx,
			caseID,
		)
	if err != nil {
		return SupportMessage{}, err
	}

	if assignment.CaseStatus ==
		"closed" {

		return SupportMessage{},
			ErrCaseClosed
	}

	if assignment.SupportActorID !=
		actorID {

		return SupportMessage{},
			ErrCaseNotOwned
	}

	var result SupportMessage

	err = tx.QueryRow(
		ctx,
		`
			INSERT INTO crm_messages (
				case_id,
				author_type,
				support_actor_id,
				visibility,
				body,
				created_at
			)
			VALUES (
				$1::uuid,
				'support',
				$2::uuid,
				$3,
				$4,
				now()
			)
			RETURNING
				id::text,
				case_id::text,
				author_type,
				visibility,
				body,
				created_at
		`,
		caseID,
		actorID,
		visibility,
		body,
	).Scan(
		&result.ID,
		&result.CaseID,
		&result.AuthorType,
		&result.Visibility,
		&result.Body,
		&result.CreatedAt,
	)
	if err != nil {
		return SupportMessage{},
			fmt.Errorf(
				"insert support message: %w",
				err,
			)
	}

	var actor TicketActor

	err = tx.QueryRow(
		ctx,
		`
			SELECT
				id::text,
				actor_code,
				actor_type,
				display_name
			FROM support_actors
			WHERE id = $1::uuid
		`,
		actorID,
	).Scan(
		&actor.ID,
		&actor.ActorCode,
		&actor.ActorType,
		&actor.DisplayName,
	)
	if err != nil {
		return SupportMessage{},
			fmt.Errorf(
				"load support message actor: %w",
				err,
			)
	}

	result.SupportActor = &actor

	nextStatus :=
		assignment.CaseStatus

	eventType :=
		"internal_note_added"

	if visibility == "customer" {
		nextStatus =
			"waiting_customer"

		eventType =
			"support_message_added"

		_, err = tx.Exec(
			ctx,
			`
				UPDATE crm_cases
				SET
					status = 'waiting_customer',
					last_message_at = now(),
					last_support_message_at = now(),
					resolved_at = NULL,
					updated_at = now()
				WHERE id = $1::uuid
			`,
			caseID,
		)
	} else {
		_, err = tx.Exec(
			ctx,
			`
				UPDATE crm_cases
				SET
					last_message_at = now(),
					last_support_message_at = now(),
					updated_at = now()
				WHERE id = $1::uuid
			`,
			caseID,
		)
	}

	if err != nil {
		return SupportMessage{},
			fmt.Errorf(
				"update CRM support message state: %w",
				err,
			)
	}

	if err :=
		insertSupportEventTx(
			ctx,
			tx,
			caseID,
			actorID,
			eventType,
			assignment.CaseStatus,
			nextStatus,
			map[string]any{
				"visibility": visibility,
			},
		); err != nil {

		return SupportMessage{},
			err
	}

	/*
		Only a customer-visible support reply
		can notify the customer.

		Internal notes remain completely
		internal and never create an
		SMS/email outbox row.
	*/
	if visibility == "customer" {
		if err :=
			r.enqueueSupportReplyNotificationTx(
				ctx,
				tx,
				caseID,
				result.ID,
			); err != nil {

			return SupportMessage{},
				err
		}
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {

		return SupportMessage{},
			fmt.Errorf(
				"commit support message: %w",
				err,
			)
	}

	return result, nil
}
