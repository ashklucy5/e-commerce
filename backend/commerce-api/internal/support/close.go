package support

import (
	"context"
	"fmt"
)

func (r *PostgresRepository) CloseCase(
	ctx context.Context,
	actorID string,
	caseID string,
) (Ticket, error) {
	tx, err :=
		r.db.Begin(
			ctx,
		)
	if err != nil {
		return Ticket{},
			fmt.Errorf(
				"begin close case transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	/*
		This locks both the active assignment and crm_cases row.

		The case remains associated with its existing assignment
		after closure so historical staff/Admin ticket reads still
		work through the current ticket query model.
	*/
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

	if assignment.SupportActorID !=
		actorID {

		return Ticket{},
			ErrCaseNotOwned
	}

	/*
		Resolution is deliberately distinct from closure.

		A resolved case can still receive a customer reply and
		reopen. Only an explicitly closed case begins the private
		support-image retention clock.
	*/
	if assignment.CaseStatus !=
		"resolved" {

		return Ticket{},
			ErrCaseNotResolved
	}

	tag, err :=
		tx.Exec(
			ctx,
			`
				UPDATE crm_cases
				SET
					status = 'closed',
					closed_at = COALESCE(
						closed_at,
						now()
					),
					updated_at = now()
				WHERE
					id = $1::uuid
					AND status = 'resolved'
			`,
			caseID,
		)
	if err != nil {
		return Ticket{},
			fmt.Errorf(
				"close CRM case: %w",
				err,
			)
	}

	if tag.RowsAffected() != 1 {
		return Ticket{},
			ErrCaseNotResolved
	}

	if err :=
		insertSupportEventTx(
			ctx,
			tx,
			caseID,
			actorID,
			"case_closed",
			"resolved",
			"closed",
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
				"commit close case: %w",
				err,
			)
	}

	return r.GetCaseForActor(
		ctx,
		actorID,
		caseID,
	)
}
