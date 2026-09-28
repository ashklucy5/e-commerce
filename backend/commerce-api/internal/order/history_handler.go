package order

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/platform/pagination"
)

func (h *Handler) List(
	c *gin.Context,
) {
	params, err :=
		pagination.Parse(
			c.Query(
				"page",
			),
			c.Query(
				"limit",
			),
		)
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_PAGINATION",

					"message": "Invalid order history pagination",
				},
			},
		)

		return
	}

	status, err :=
		NormalizeOrderHistoryStatus(
			c.Query(
				"status",
			),
		)
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_ORDER_STATUS",

					"message": "Invalid order status",
				},
			},
		)

		return
	}

	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{
				"error": gin.H{
					"code": "AUTHENTICATION_REQUIRED",

					"message": "Customer authentication is required",
				},
			},
		)

		return
	}

	result, err :=
		h.service.ListForCustomer(
			c.Request.Context(),
			customerID,
			params,
			status,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			ErrCustomerAuthenticationRequired,
		):
			c.JSON(
				http.StatusUnauthorized,
				gin.H{
					"error": gin.H{
						"code": "AUTHENTICATION_REQUIRED",

						"message": "Customer authentication is required",
					},
				},
			)

		case errors.Is(
			err,
			ErrInvalidOrderHistoryStatus,
		):
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": gin.H{
						"code": "INVALID_ORDER_STATUS",

						"message": "Invalid order status",
					},
				},
			)

		default:
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": gin.H{
						"code": "INTERNAL_ERROR",

						"message": "Unable to load order history",
					},
				},
			)
		}

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
