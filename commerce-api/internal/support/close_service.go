package support

import (
	"context"
	"fmt"
)

func (s *Service) Close(
	ctx context.Context,
	staffID string,
	caseID string,
) (Ticket, error) {
	if !validSupportUUID(
		caseID,
	) {
		return Ticket{},
			ErrCaseNotFound
	}

	actor, err :=
		s.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return Ticket{}, err
	}

	return s.repository.CloseCase(
		ctx,
		actor.ID,
		caseID,
	)
}

// Close exposes the same resolved -> closed transition through
// the Admin CRM surface.
//
// Queue-scoped Support Agents/Supervisors continue through the
// normal support Service so their queue membership rules remain
// unchanged.
//
// Administrator/Super Admin principals can operate across queues,
// but the mutation is still attributed to their real human
// support actor and the case must be assigned to that actor.
func (s *AdminService) Close(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	caseID string,
) (Ticket, error) {
	if queueScoped {
		return s.regular.Close(
			ctx,
			staffID,
			caseID,
		)
	}

	if !validSupportUUID(
		caseID,
	) {
		return Ticket{},
			ErrCaseNotFound
	}

	actor, err :=
		s.ensureAdminActor(
			ctx,
			staffID,
		)
	if err != nil {
		return Ticket{}, err
	}

	tx, err :=
		s.db.Begin(
			ctx,
		)
	if err != nil {
		return Ticket{},
			fmt.Errorf(
				"begin Admin CRM close transaction: %w",
				err,
			)
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	assignment, err :=
		s.repo.lockActiveAssignmentTx(
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
		actor.ID {

		return Ticket{},
			ErrCaseNotOwned
	}

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
				"close Admin CRM case: %w",
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
			actor.ID,
			"case_closed",
			"resolved",
			"closed",
			map[string]any{
				"queue_code": assignment.QueueCode,

				"source": "admin_panel",
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
				"commit Admin CRM close: %w",
				err,
			)
	}

	/*
		Global Admin reads intentionally use the Admin read
		path rather than GetCaseForActor because a global
		Admin support actor does not need queue membership.
	*/
	return s.GetCase(
		ctx,
		staffID,
		false,
		caseID,
	)
}
