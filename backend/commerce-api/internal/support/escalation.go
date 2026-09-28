package support

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) EscalateCase(
	ctx context.Context,
	actorID string,
	caseID string,
	queueCode string,
) (EscalationResult, error) {
	tx, err := r.db.Begin(
		ctx,
	)
	if err != nil {
		return EscalationResult{},
			fmt.Errorf(
				"begin escalation transaction: %w",
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
		return EscalationResult{}, err
	}

	if assignment.CaseStatus ==
		"closed" {
		return EscalationResult{},
			ErrCaseClosed
	}

	if assignment.SupportActorID !=
		actorID {
		return EscalationResult{},
			ErrCaseNotOwned
	}

	var targetQueueID string

	err =
		tx.QueryRow(
			ctx,
			`
				SELECT id::text
				FROM support_queues
				WHERE
					code = $1
					AND status = 'active'
			`,
			queueCode,
		).Scan(
			&targetQueueID,
		)
	if err == pgx.ErrNoRows {
		return EscalationResult{},
			ErrInvalidQueue
	}
	if err != nil {
		return EscalationResult{},
			fmt.Errorf(
				"load escalation queue: %w",
				err,
			)
	}

	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE support_case_assignments
				SET
					released_at = now(),
					release_reason = $2
				WHERE id = $1::uuid
			`,
			assignment.AssignmentID,
			"escalated to "+queueCode,
		)
	if err != nil {
		return EscalationResult{},
			fmt.Errorf(
				"release escalated assignment: %w",
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
					assigned_at
				)
				VALUES (
					$1::uuid,
					$2::uuid,
					NULL,
					$3::uuid,
					'escalated',
					now()
				)
			`,
			caseID,
			targetQueueID,
			actorID,
		)
	if err != nil {
		return EscalationResult{},
			fmt.Errorf(
				"create escalated assignment: %w",
				err,
			)
	}

	_, err =
		tx.Exec(
			ctx,
			`
				UPDATE crm_cases
				SET
					status = 'waiting_support',
					resolved_at = NULL,
					updated_at = now()
				WHERE id = $1::uuid
			`,
			caseID,
		)
	if err != nil {
		return EscalationResult{},
			fmt.Errorf(
				"update escalated CRM case: %w",
				err,
			)
	}

	if err :=
		insertSupportEventTx(
			ctx,
			tx,
			caseID,
			actorID,
			"case_escalated",
			assignment.CaseStatus,
			"waiting_support",
			map[string]any{
				"from_queue": assignment.QueueCode,

				"to_queue": queueCode,
			},
		); err != nil {
		return EscalationResult{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return EscalationResult{},
			fmt.Errorf(
				"commit escalation: %w",
				err,
			)
	}

	return EscalationResult{
		CaseID: caseID,

		Status: "waiting_support",

		QueueCode: queueCode,
	}, nil
}
