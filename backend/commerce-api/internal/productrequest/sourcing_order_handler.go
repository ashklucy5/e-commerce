package productrequest

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/order"
)

func (h *Handler) PlaceSourcingOrder(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeSourcingOrderCustomerError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	var request PlaceSourcingOrderRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeSourcingOrderCustomerError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.PlaceSourcingOrder(
			c.Request.Context(),
			customerID,
			strings.TrimSpace(
				c.Param(
					"id",
				),
			),
			request,
		)
	if err != nil {
		writeSourcingOrderCustomerError(
			c,
			err,
		)

		return
	}

	status :=
		http.StatusCreated

	if !result.Created {
		status =
			http.StatusOK
	}

	c.JSON(
		status,
		gin.H{
			"data": result,
		},
	)
}

func writeSourcingOrderCustomerError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		auth.ErrInvalidAccessToken,
	):
		writeJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

	case errors.Is(
		err,
		ErrInvalidRequest,
	):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SOURCING_ORDER",
			"Invalid sourcing order input",
		)

	case errors.Is(
		err,
		ErrNotFound,
	):
		writeJSONError(
			c,
			http.StatusNotFound,
			"PRODUCT_REQUEST_NOT_FOUND",
			"Product request was not found",
		)

	case errors.Is(
		err,
		ErrConfirmationNotFound,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"SOURCING_ORDER_NOT_READY",
			"The sourcing agreement is not ready to be converted into an order",
		)

	case errors.Is(
		err,
		ErrOfferConflict,
	),
		errors.Is(
			err,
			ErrConversationClosed,
		):

		writeJSONError(
			c,
			http.StatusConflict,
			"SOURCING_ORDER_NOT_ACTIONABLE",
			"This sourcing agreement cannot be converted into an order in its current state",
		)

	case errors.Is(
		err,
		order.ErrCustomerDetailsRequired,
	):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"CUSTOMER_DETAILS_REQUIRED",
			"Customer name and phone are required",
		)

	case errors.Is(
		err,
		order.ErrShippingDetailsRequired,
	):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"SHIPPING_DETAILS_REQUIRED",
			"Shipping address details are required",
		)

	case errors.Is(
		err,
		order.ErrInvalidPaymentMethod,
	):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_PAYMENT_METHOD",
			"The selected payment method is not available",
		)

	default:
		// Commercial-validation failures are intentionally not
		// exposed as customer-editable errors here.
		//
		// Quantity, MOQ, prices, currency and total come from the
		// immutable finalized sourcing confirmation, so failures in
		// those fields represent server/domain state rather than
		// untrusted customer input.
		writeJSONError(
			c,
			http.StatusInternalServerError,
			"SOURCING_ORDER_FAILED",
			"Unable to create sourcing order",
		)
	}
}
