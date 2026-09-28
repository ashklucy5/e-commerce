package support

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func DefaultQueueCode(
	caseType string,
) string {
	switch caseType {
	case "bulk_stock_request":
		return "bulk_sales"
	case "order_issue":
		return "orders"
	case "payment_issue":
		return "payments"
	case "return_issue":
		return "returns_refunds"
	case "delivery_issue":
		return "delivery"
	case "complaint":
		return "complaints"
	default:
		return "general"
	}
}

func AssignInitialCaseTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	queueCode string,
) error {
	tag, err := tx.Exec(
		ctx,
		`
			INSERT INTO support_case_assignments (
				case_id,
				queue_id,
				support_actor_id,
				assigned_by_actor_id,
				assignment_type,
				assigned_at
			)
			SELECT
				$1::uuid,
				q.id,
				NULL,
				NULL,
				'automatic',
				now()
			FROM support_queues q
			WHERE
				q.code = $2
				AND q.status = 'active'
		`,
		caseID,
		queueCode,
	)
	if err != nil {
		return fmt.Errorf(
			"create initial support assignment: %w",
			err,
		)
	}

	if tag.RowsAffected() != 1 {
		return ErrAssignmentQueueUnavailable
	}

	return nil
}

func (r *PostgresRepository) ClaimCase(
	ctx context.Context,
	actorID string,
	caseID string,
) (Ticket, error) {
	tx, err := r.db.Begin(
		ctx,
	)
	if err != nil {
		return Ticket{},
			fmt.Errorf(
				"begin claim case transaction: %w",
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
		return Ticket{}, err
	}

	if assignment.CaseStatus ==
		"closed" {
		return Ticket{},
			ErrCaseClosed
	}

	allowed, err :=
		actorHasQueueAccessTx(
			ctx,
			tx,
			actorID,
			assignment.QueueID,
		)
	if err != nil {
		return Ticket{}, err
	}

	if !allowed {
		return Ticket{},
			ErrCaseNotFound
	}

	if assignment.SupportActorID != "" {
		if assignment.SupportActorID ==
			actorID {
			if err :=
				tx.Commit(
					ctx,
				); err != nil {
				return Ticket{},
					err
			}

			return r.GetCaseForActor(
				ctx,
				actorID,
				caseID,
			)
		}

		return Ticket{},
			ErrCaseAlreadyClaimed
	}

	var maxCases int

	err =
		tx.QueryRow(
			ctx,
			`
				SELECT max_active_cases
				FROM support_actors
				WHERE
					id = $1::uuid
					AND status = 'active'
				FOR UPDATE
			`,
			actorID,
		).Scan(
			&maxCases,
		)
	if err != nil {
		return Ticket{},
			fmt.Errorf(
				"load support actor capacity: %w",
				err,
			)
	}

	var activeCases int

	err =
		tx.QueryRow(
			ctx,
			`
				SELECT COUNT(*)
				FROM support_case_assignments a
				JOIN crm_cases c
					ON c.id = a.case_id
				WHERE
					a.support_actor_id = $1::uuid
					AND a.released_at IS NULL
					AND c.status IN (
						'waiting_support',
						'waiting_customer'
					)
			`,
			actorID,
		).Scan(
			&activeCases,
		)
	if err != nil {
		return Ticket{},
			fmt.Errorf(
				"count support actor cases: %w",
				err,
			)
	}

	if activeCases >= maxCases {
		return Ticket{},
			ErrActorCapacity
	}

	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE support_case_assignments
				SET
					released_at = now(),
					release_reason = 'claimed'
				WHERE id = $1::uuid
			`,
			assignment.AssignmentID,
		)
	if err != nil {
		return Ticket{},
			fmt.Errorf(
				"release queued assignment: %w",
				err,
			)
	}

	_, err =
		tx.Exec(
			ctx,
			`
				INSERT INTO support_case_assignments (
					case_id,
					queue_id,
					support_actor_id,
					assigned_by_actor_id,
					assignment_type,
					assigned_at,
					accepted_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					$3::uuid,
					$3::uuid,
					'claimed',
					now(),
					now()
				)
			`,
			caseID,
			assignment.QueueID,
			actorID,
		)
	if err != nil {
		return Ticket{},
			fmt.Errorf(
				"create claimed assignment: %w",
				err,
			)
	}

	if err :=
		insertSupportEventTx(
			ctx,
			tx,
			caseID,
			actorID,
			"case_claimed",
			assignment.CaseStatus,
			assignment.CaseStatus,
			map[string]any{
				"queue_code": assignment.QueueCode,
			},
		); err != nil {
		return Ticket{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Ticket{},
			fmt.Errorf(
				"commit case claim: %w",
				err,
			)
	}

	return r.GetCaseForActor(
		ctx,
		actorID,
		caseID,
	)
}
