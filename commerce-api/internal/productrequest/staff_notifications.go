package productrequest

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"project.local/commerce-api/internal/notification"
)

func (r *Repository) enqueueStaffProductRequestCreatedTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	caseID string,
	requestNumber string,
	productName string,
	requestedQuantity int,
) error {
	if err :=
		notification.EnqueueStaffEventTx(
			ctx,
			tx,
			notification.StaffEventRequest{
				DedupeKey: "staff:sourcing:" +
					requestID +
					":created",

				Category: notification.StaffCategorySourcing,

				EventType: notification.StaffEventProductRequestNew,

				Priority: notification.StaffPriorityAttention,

				Title: "New product request",

				Message: fmt.Sprintf(
					"Product request %s was submitted for %s.",
					requestNumber,
					productName,
				),

				ActionURL: "/admin/product-requests",

				EntityType: "product_request",

				EntityID: requestID,

				RequiredPermission: "admin.sourcing.read",

				Metadata: map[string]any{
					"case_id": caseID,

					"request_number": requestNumber,

					"requested_product_name": productName,

					"requested_quantity": requestedQuantity,
				},
			},
		); err != nil {

		return fmt.Errorf(
			"enqueue new product request staff notification: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) enqueueStaffProductRequestCustomerMessageTx(
	ctx context.Context,
	tx pgx.Tx,
	requestID string,
	caseID string,
	messageID string,
) error {
	var (
		requestNumber string
		productName   string
	)

	if err :=
		tx.QueryRow(
			ctx,
			`
				SELECT
					request_number,
					requested_product_name
				FROM product_sourcing_requests
				WHERE id = $1::uuid
			`,
			requestID,
		).Scan(
			&requestNumber,
			&productName,
		); err != nil {

		return fmt.Errorf(
			"load product request notification context: %w",
			err,
		)
	}

	if err :=
		notification.EnqueueStaffEventTx(
			ctx,
			tx,
			notification.StaffEventRequest{
				DedupeKey: "staff:sourcing:" +
					requestID +
					":customer-message:" +
					messageID,

				Category: notification.StaffCategorySourcing,

				EventType: notification.StaffEventProductRequestReply,

				Priority: notification.StaffPriorityAttention,

				Title: "Product request reply",

				Message: fmt.Sprintf(
					"Customer replied to product request %s.",
					requestNumber,
				),

				ActionURL: "/admin/product-requests",

				EntityType: "product_request",

				EntityID: requestID,

				RequiredPermission: "admin.sourcing.read",

				Metadata: map[string]any{
					"case_id": caseID,

					"message_id": messageID,

					"request_number": requestNumber,

					"requested_product_name": productName,
				},
			},
		); err != nil {

		return fmt.Errorf(
			"enqueue product request customer reply staff notification: %w",
			err,
		)
	}

	return nil
}