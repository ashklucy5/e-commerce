package admin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

func (h *Handler) Orders(
	c *gin.Context,
) {
	params, ok := adminOperationalPagination(c)
	if !ok {
		return
	}

	status := strings.ToLower(
		strings.TrimSpace(
			c.Query("status"),
		),
	)

	if !validOrderStatusFilter(status) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ORDER_STATUS",
				"Invalid order status",
			),
		)

		return
	}

	paymentStatus := strings.ToLower(
		strings.TrimSpace(
			c.Query("payment_status"),
		),
	)

	if !validOrderPaymentStatusFilter(
		paymentStatus,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ORDER_PAYMENT_STATUS",
				"Invalid order payment status",
			),
		)

		return
	}

	query, ok := adminOperationalQuery(c)
	if !ok {
		return
	}

	result, err := h.service.ListOrders(
		c.Request.Context(),
		params,
		OrderReadFilter{
			Status:        status,
			PaymentStatus: paymentStatus,
			Query:         query,
		},
	)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(err),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result.Items,
			"meta": result.Meta,
		},
	)
}

func (h *Handler) Order(
	c *gin.Context,
) {
	orderID := strings.TrimSpace(
		c.Param("order_id"),
	)

	if !platformvalidation.IsUUID(orderID) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ORDER_ID",
				"order_id must be a valid UUID",
			),
		)

		return
	}

	result, err := h.service.GetOrder(
		c.Request.Context(),
		orderID,
	)

	if errors.Is(
		err,
		ErrAdminOrderNotFound,
	) {
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"ORDER_NOT_FOUND",
				"Order not found",
			),
		)

		return
	}

	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(err),
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

func (h *Handler) Returns(
	c *gin.Context,
) {
	params, ok := adminOperationalPagination(c)
	if !ok {
		return
	}

	status := strings.ToLower(
		strings.TrimSpace(
			c.Query("status"),
		),
	)

	if !validReturnStatusFilter(status) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_RETURN_STATUS",
				"Invalid return status",
			),
		)

		return
	}

	query, ok := adminOperationalQuery(c)
	if !ok {
		return
	}

	result, err := h.service.ListReturns(
		c.Request.Context(),
		params,
		ReturnReadFilter{
			Status: status,
			Query:  query,
		},
	)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(err),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result.Items,
			"meta": result.Meta,
		},
	)
}

func (h *Handler) Return(
	c *gin.Context,
) {
	returnID := strings.TrimSpace(
		c.Param("return_id"),
	)

	if !platformvalidation.IsUUID(returnID) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_RETURN_ID",
				"return_id must be a valid UUID",
			),
		)

		return
	}

	result, err := h.service.GetReturn(
		c.Request.Context(),
		returnID,
	)

	if errors.Is(
		err,
		ErrAdminReturnNotFound,
	) {
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"RETURN_NOT_FOUND",
				"Return not found",
			),
		)

		return
	}

	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(err),
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

func (h *Handler) Payments(
	c *gin.Context,
) {
	params, ok := adminOperationalPagination(c)
	if !ok {
		return
	}

	status := strings.ToLower(
		strings.TrimSpace(
			c.Query("status"),
		),
	)

	if !validPaymentStatusFilter(status) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PAYMENT_STATUS",
				"Invalid payment status",
			),
		)

		return
	}

	provider := strings.ToLower(
		strings.TrimSpace(
			c.Query("provider"),
		),
	)

	if !validPaymentProviderFilter(provider) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PAYMENT_PROVIDER",
				"Invalid payment provider",
			),
		)

		return
	}

	query, ok := adminOperationalQuery(c)
	if !ok {
		return
	}

	result, err := h.service.ListPayments(
		c.Request.Context(),
		params,
		PaymentReadFilter{
			Status:   status,
			Provider: provider,
			Query:    query,
		},
	)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(err),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result.Items,
			"meta": result.Meta,
		},
	)
}

func (h *Handler) Payment(
	c *gin.Context,
) {
	paymentID := strings.TrimSpace(
		c.Param("payment_id"),
	)

	if !platformvalidation.IsUUID(paymentID) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PAYMENT_ID",
				"payment_id must be a valid UUID",
			),
		)

		return
	}

	result, err := h.service.GetPayment(
		c.Request.Context(),
		paymentID,
	)

	if errors.Is(
		err,
		ErrAdminPaymentNotFound,
	) {
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"PAYMENT_NOT_FOUND",
				"Payment not found",
			),
		)

		return
	}

	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(err),
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

func adminOperationalPagination(
	c *gin.Context,
) (
	platformpagination.Params,
	bool,
) {
	params, err := platformpagination.Parse(
		c.Query("page"),
		c.Query("limit"),
	)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PAGINATION",
				"Invalid pagination",
			),
		)

		return platformpagination.Params{},
			false
	}

	return params,
		true
}

func adminOperationalQuery(
	c *gin.Context,
) (
	string,
	bool,
) {
	query := strings.TrimSpace(
		c.Query("q"),
	)

	if len([]rune(query)) > 100 {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_SEARCH_QUERY",
				"q cannot exceed 100 characters",
			),
		)

		return "",
			false
	}

	return query,
		true
}

func validOrderStatusFilter(
	value string,
) bool {
	switch value {
	case "",
		"pending_payment",
		"confirmed",
		"processing",
		"shipped",
		"delivered",
		"completed",
		"payment_expired",
		"cancelled":
		return true

	default:
		return false
	}
}

func validOrderPaymentStatusFilter(
	value string,
) bool {
	switch value {
	case "",
		"pending",
		"paid",
		"failed",
		"expired",
		"cod_pending",
		"cod_collected",
		"refunded":
		return true

	default:
		return false
	}
}

func validReturnStatusFilter(
	value string,
) bool {
	switch value {
	case "",
		"requested",
		"approved",
		"rejected",
		"received",
		"inspected",
		"completed",
		"cancelled":
		return true

	default:
		return false
	}
}

func validPaymentStatusFilter(
	value string,
) bool {
	switch value {
	case "",
		"pending",
		"succeeded",
		"failed",
		"expired",
		"refunded":
		return true

	default:
		return false
	}
}

func validPaymentProviderFilter(
	value string,
) bool {
	switch value {
	case "",
		"bkash",
		"nagad",
		"rocket",
		"bank_transfer":
		return true

	default:
		return false
	}
}
