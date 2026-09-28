package support

import (
	"context"
	"fmt"
)

// AttachmentActorForUpload resolves the support actor representing the
// authenticated Admin staff member and verifies that the actor currently
// owns the CRM case.
//
// Support Agent/Supervisor principals remain queue-scoped. Administrator
// and Super Admin principals retain their existing cross-queue behavior.
func (s *AdminService) AttachmentActorForUpload(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	caseID string,
) (Actor, error) {
	if !validSupportUUID(caseID) {
		return Actor{}, ErrCaseNotFound
	}

	var (
		actor  Actor
		ticket Ticket
		err    error
	)

	if queueScoped {
		actor, err =
			s.regular.getActiveActor(
				ctx,
				staffID,
			)
		if err != nil {
			return Actor{}, err
		}

		ticket, err =
			s.repo.GetCaseForActor(
				ctx,
				actor.ID,
				caseID,
			)
		if err != nil {
			return Actor{}, err
		}
	} else {
		actor, err =
			s.ensureAdminActor(
				ctx,
				staffID,
			)
		if err != nil {
			return Actor{}, err
		}

		ticket, err =
			s.GetCase(
				ctx,
				staffID,
				false,
				caseID,
			)
		if err != nil {
			return Actor{}, err
		}
	}

	if ticket.Status == "closed" {
		return Actor{}, ErrCaseClosed
	}

	if ticket.Assignment.AssignedActor == nil ||
		ticket.Assignment.AssignedActor.ID !=
			actor.ID {

		return Actor{},
			ErrCaseNotOwned
	}

	return actor, nil
}

// AttachmentActor resolves the support actor represented by an Admin
// principal without requiring a case.
//
// This is used when completing a previously created private upload.
func (s *AdminService) AttachmentActor(
	ctx context.Context,
	staffID string,
	queueScoped bool,
) (Actor, error) {
	if queueScoped {
		return s.regular.getActiveActor(
			ctx,
			staffID,
		)
	}

	return s.ensureAdminActor(
		ctx,
		staffID,
	)
}

// CanAccessAttachment verifies that an Admin principal is authorized to
// view the CRM message/case that owns an attachment.
//
// Global Administrators/Super Admins may access any CRM attachment covered
// by their admin.crm.read permission.
//
// Queue-scoped Support Agents/Supervisors must still belong to the active
// queue for the owning case.
//
// Internal-note attachments are intentionally visible here because this is
// the staff/Admin surface rather than the customer surface.
func (s *AdminService) CanAccessAttachment(
	ctx context.Context,
	staffID string,
	queueScoped bool,
	attachmentID string,
) (bool, error) {
	if !validSupportUUID(
		attachmentID,
	) {
		return false, nil
	}

	if !queueScoped {
		var allowed bool

		err :=
			s.db.QueryRow(
				ctx,
				`
					SELECT EXISTS (
						SELECT 1
						FROM crm_support_attachments a
						JOIN crm_messages m
							ON m.id = a.message_id
							AND m.case_id = a.case_id
						JOIN crm_cases c
							ON c.id = a.case_id
						WHERE
							a.id = $1::uuid
					)
				`,
				attachmentID,
			).Scan(
				&allowed,
			)
		if err != nil {
			return false,
				fmt.Errorf(
					"authorize Admin CRM attachment: %w",
					err,
				)
		}

		return allowed, nil
	}

	actor, err :=
		s.regular.getActiveActor(
			ctx,
			staffID,
		)
	if err != nil {
		return false, err
	}

	var allowed bool

	err =
		s.db.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM crm_support_attachments a
					JOIN crm_messages m
						ON m.id = a.message_id
						AND m.case_id = a.case_id
					JOIN crm_cases c
						ON c.id = a.case_id
					JOIN support_case_assignments ca
						ON ca.case_id = c.id
						AND ca.released_at IS NULL
					JOIN support_queue_members qm
						ON qm.queue_id = ca.queue_id
						AND qm.support_actor_id = $2::uuid
					WHERE
						a.id = $1::uuid
				)
			`,
			attachmentID,
			actor.ID,
		).Scan(
			&allowed,
		)
	if err != nil {
		return false,
			fmt.Errorf(
				"authorize queue-scoped Admin CRM attachment: %w",
				err,
			)
	}

	return allowed, nil
}
