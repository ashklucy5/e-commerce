package admin

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/order"
	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

func (h *Handler) OrderInvoice(
	c *gin.Context,
) {
	orderID :=
		strings.TrimSpace(
			c.Param(
				"order_id",
			),
		)

	if !platformvalidation.IsUUID(
		orderID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ORDER_ID",
				"order_id must be a valid UUID",
			),
		)

		return
	}

	result, err :=
		h.service.GetOrderInvoice(
			c.Request.Context(),
			orderID,
		)

	if errors.Is(
		err,
		order.ErrInvoiceNotFound,
	) ||
		errors.Is(
			err,
			order.ErrOrderNotFound,
		) {

		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"INVOICE_NOT_FOUND",
				"Invoice not found",
			),
		)

		return
	}

	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) ResendOrderInvoice(
	c *gin.Context,
) {
	orderID :=
		strings.TrimSpace(
			c.Param(
				"order_id",
			),
		)

	if !platformvalidation.IsUUID(
		orderID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ORDER_ID",
				"order_id must be a valid UUID",
			),
		)

		return
	}

	idempotencyKey :=
		strings.TrimSpace(
			c.GetHeader(
				"Idempotency-Key",
			),
		)

	if idempotencyKey == "" {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"IDEMPOTENCY_KEY_REQUIRED",
				"Idempotency-Key header is required",
			),
		)

		return
	}

	result, err :=
		h.service.ResendOrderInvoice(
			c.Request.Context(),
			orderID,
			idempotencyKey,
		)

	if errors.Is(
		err,
		order.ErrInvalidInvoiceResendKey,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_IDEMPOTENCY_KEY",
				"Invalid Idempotency-Key header",
			),
		)

		return
	}

	if errors.Is(
		err,
		order.ErrInvoiceNotFound,
	) ||
		errors.Is(
			err,
			order.ErrOrderNotFound,
		) {

		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"INVOICE_NOT_FOUND",
				"Invoice not found",
			),
		)

		return
	}

	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	/*
		The email has been durably queued in the notification
		outbox. HTTP 202 does not claim that an external provider
		has already delivered it.
	*/
	c.JSON(
		http.StatusAccepted,
		gin.H{
			"data": result,
		},
	)
}

func (s *Service) GetOrderInvoice(
	ctx context.Context,
	orderID string,
) (
	order.Invoice,
	error,
) {
	repository :=
		order.NewRepository(
			s.db,
		)

	return repository.GetInvoiceByOrderID(
		ctx,
		orderID,
	)
}

func (s *Service) ResendOrderInvoice(
	ctx context.Context,
	orderID string,
	idempotencyKey string,
) (
	order.InvoiceResendResult,
	error,
) {
	repository :=
		order.NewRepository(
			s.db,
		)

	return repository.ResendInvoiceNotifications(
		ctx,
		orderID,
		idempotencyKey,
	)
}
