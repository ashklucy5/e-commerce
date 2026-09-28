package returns

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/platform/pagination"
)

func (h *Handler) ListForCustomer(
	c *gin.Context,
) {
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

					"message": "Invalid return history pagination",
				},
			},
		)

		return
	}

	status, err :=
		NormalizeReturnHistoryStatus(
			c.Query(
				"status",
			),
		)
	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_RETURN_STATUS",

					"message": "Invalid return status",
				},
			},
		)

		return
	}

	result, err :=
		h.service.ListHistoryForCustomer(
			c.Request.Context(),
			customerID,
			params,
			status,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			ErrReturnHistoryAuthenticationRequired,
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
			ErrInvalidReturnHistoryStatus,
		):
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": gin.H{
						"code": "INVALID_RETURN_STATUS",

						"message": "Invalid return status",
					},
				},
			)

		default:
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": gin.H{
						"code": "INTERNAL_ERROR",

						"message": "Unable to load return history",
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
