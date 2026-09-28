package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func EnqueueCustomerInboxTx(
	ctx context.Context,
	tx pgx.Tx,
	request CustomerInboxRequest,
) error {
	if tx == nil {
		return fmt.Errorf("notification transaction is required")
	}

	repository := &Repository{}
	_, _, err := repository.InsertCustomerInboxTx(ctx, tx, request)
	return err
}

func EnqueueStaffEventTx(
	ctx context.Context,
	tx pgx.Tx,
	request StaffEventRequest,
) error {
	if tx == nil {
		return fmt.Errorf("notification transaction is required")
	}

	repository := &Repository{}
	_, err := repository.InsertStaffEventTx(ctx, tx, request)
	return err
}

func customerInboxPresentation(
	request CustomerEventRequest,
) (title string, actionURL string) {
	if strings.TrimSpace(request.Title) != "" {
		title = strings.TrimSpace(request.Title)
	}
	if strings.TrimSpace(request.ActionURL) != "" {
		actionURL = strings.TrimSpace(request.ActionURL)
	}

	if title == "" {
		switch request.EventType {
		case EventOrderPlaced:
			title = "Order received"

		case EventOrderProcessing:
			title = "Order is being prepared"

		case EventPaymentSucceeded:
			title = "Payment confirmed"

		case EventDeliverySent:
			title = "Order shipped"

		case EventDeliveryUpdated:
			title = "Delivery update"

		case EventDeliveryComplete:
			title = "Order delivered"

		case EventSupportReply:
			title = "Support replied"

		case EventInvoiceIssued:
			title = "Invoice ready"

		case EventInvoiceResent:
			title = "Invoice resent"

		default:
			title = "Account update"
		}
	}

	if actionURL == "" {
		switch {
		case request.OrderID != "":
			actionURL = "/account/orders/" + request.OrderID

		case request.CaseID != "":
			actionURL = "/account/support"

		default:
			actionURL = "/account/notifications"
		}
	}

	return title, actionURL
}
