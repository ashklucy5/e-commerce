package support

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/notification"
)

type supportNotificationSnapshot struct {
	CustomerID string

	CaseNumber string

	Phone string
	Email string
}

func (r *PostgresRepository) enqueueSupportReplyNotificationTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
	messageID string,
) error {
	snapshot, err :=
		r.supportNotificationSnapshotTx(
			ctx,
			tx,
			caseID,
		)
	if err != nil {
		return err
	}

	message :=
		fmt.Sprintf(
			"You have a new support reply for case %s.",
			snapshot.CaseNumber,
		)

	if err :=
		notification.EnqueueCustomerEventTx(
			ctx,
			tx,
			notification.CustomerEventRequest{
				BaseDedupeKey: "support:" +
					caseID +
					":message:" +
					messageID,

				Category: notification.CategorySupport,

				EventType: notification.EventSupportReply,

				CustomerID: snapshot.CustomerID,

				CaseID: caseID,

				Phone: snapshot.Phone,

				Email: snapshot.Email,

				TemplateKey: notification.TemplateSupportReply,

				Message: message,

				Payload: map[string]any{
					"case_number": snapshot.CaseNumber,
				},
			},
		); err != nil {

		return fmt.Errorf(
			"enqueue support reply notification: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresRepository) supportNotificationSnapshotTx(
	ctx context.Context,
	tx pgx.Tx,
	caseID string,
) (
	supportNotificationSnapshot,
	error,
) {
	var result supportNotificationSnapshot

	err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					c.customer_id::text,

					c.case_number,

					customer.phone,

					COALESCE(
						customer.email,
						''
					)

				FROM crm_cases c

				JOIN customers customer
					ON customer.id =
						c.customer_id

				WHERE
					c.id =
						$1::uuid
			`,
			caseID,
		).Scan(
			&result.CustomerID,
			&result.CaseNumber,
			&result.Phone,
			&result.Email,
		)
	if err != nil {
		return supportNotificationSnapshot{},
			fmt.Errorf(
				"load support notification snapshot: %w",
				err,
			)
	}

	return result,
		nil
}
