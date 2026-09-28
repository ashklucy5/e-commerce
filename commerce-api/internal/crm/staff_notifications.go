package crm

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/notification"
)

func (r *Repository) enqueueStaffCaseCreatedTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	caseNumber string,
	caseType string,
	subject string,
	priority string,
) error {
	staffPriority :=
		notification.StaffPriorityAttention

	if priority == "urgent" {
		staffPriority =
			notification.StaffPriorityCritical
	}

	if err :=
		notification.EnqueueStaffEventTx(
			ctx,
			tx,
			notification.StaffEventRequest{
				DedupeKey: "staff:support:" +
					caseID +
					":created",

				Category: notification.StaffCategorySupport,

				EventType: notification.StaffEventSupportCaseNew,

				Priority: staffPriority,

				Title: "New support case",

				Message: fmt.Sprintf(
					"Support case %s was opened: %s",
					caseNumber,
					subject,
				),

				ActionURL: "/admin/support",

				EntityType: "crm_case",

				EntityID: caseID,

				RequiredPermission: "admin.crm.read",

				Metadata: map[string]any{
					"case_number": caseNumber,

					"case_type": caseType,

					"priority": priority,
				},
			},
		); err != nil {

		return fmt.Errorf(
			"enqueue new support case staff notification: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) enqueueStaffCustomerReplyTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	messageID string,
) error {
	var (
		caseNumber string
		subject    string
		priority   string
	)

	if err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					case_number,
					subject,
					priority
				FROM crm_cases
				WHERE id = $1::uuid
			`,
			caseID,
		).Scan(
			&caseNumber,
			&subject,
			&priority,
		); err != nil {

		return fmt.Errorf(
			"load support notification context: %w",
			err,
		)
	}

	staffPriority :=
		notification.StaffPriorityAttention

	if priority == "urgent" {
		staffPriority =
			notification.StaffPriorityCritical
	}

	if err :=
		notification.EnqueueStaffEventTx(
			ctx,
			tx,
			notification.StaffEventRequest{
				DedupeKey: "staff:support:" +
					caseID +
					":customer-message:" +
					messageID,

				Category: notification.StaffCategorySupport,

				EventType: notification.StaffEventSupportCustomerReply,

				Priority: staffPriority,

				Title: "Customer replied",

				Message: fmt.Sprintf(
					"Customer replied to support case %s.",
					caseNumber,
				),

				ActionURL: "/admin/support",

				EntityType: "crm_case",

				EntityID: caseID,

				RequiredPermission: "admin.crm.read",

				Metadata: map[string]any{
					"case_number": caseNumber,

					"subject": subject,

					"message_id": messageID,

					"priority": priority,
				},
			},
		); err != nil {

		return fmt.Errorf(
			"enqueue customer support reply staff notification: %w",
			err,
		)
	}

	return nil
}
