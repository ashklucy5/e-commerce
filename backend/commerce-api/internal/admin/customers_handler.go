package admin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

type customerStatusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) Customers(
	c *gin.Context,
) {
	params, ok :=
		adminOperationalPagination(c)
	if !ok {
		return
	}

	status :=
		strings.ToLower(
			strings.TrimSpace(
				c.Query("status"),
			),
		)

	if status != "" &&
		status != "active" &&
		status != "disabled" {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_CUSTOMER_STATUS",
				"status must be active or disabled",
			),
		)

		return
	}

	query, ok :=
		adminOperationalQuery(c)
	if !ok {
		return
	}

	queryID := ""

	if platformvalidation.IsUUID(
		query,
	) {
		queryID =
			query
	}

	result, err :=
		h.service.ListCustomers(
			c.Request.Context(),
			params,
			CustomerReadFilter{
				Status: status,

				Query: query,

				QueryID: queryID,
			},
		)
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
			"data": result.Items,

			"meta": result.Meta,
		},
	)
}

func (h *Handler) Customer(
	c *gin.Context,
) {
	customerID :=
		strings.TrimSpace(
			c.Param(
				"customer_id",
			),
		)

	if !platformvalidation.IsUUID(
		customerID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_CUSTOMER_ID",
				"customer_id must be a valid UUID",
			),
		)

		return
	}

	result, err :=
		h.service.GetCustomer(
			c.Request.Context(),
			customerID,
		)

	if errors.Is(
		err,
		ErrAdminCustomerNotFound,
	) {
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"CUSTOMER_NOT_FOUND",
				"Customer not found",
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

func (h *Handler) UpdateCustomerStatus(
	c *gin.Context,
) {
	customerID :=
		strings.TrimSpace(
			c.Param(
				"customer_id",
			),
		)

	if !platformvalidation.IsUUID(
		customerID,
	) {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_CUSTOMER_ID",
				"customer_id must be a valid UUID",
			),
		)

		return
	}

	var request customerStatusRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_CUSTOMER_STATUS_REQUEST",
				"Invalid customer status request",
			),
		)

		return
	}

	status :=
		strings.ToLower(
			strings.TrimSpace(
				request.Status,
			),
		)

	if status != "active" &&
		status != "disabled" {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_CUSTOMER_STATUS",
				"status must be active or disabled",
			),
		)

		return
	}

	metadata, ok :=
		adminActionMetadataFromContext(c)
	if !ok {
		platformerrors.Write(
			c,
			platformerrors.Unauthorized(
				"ADMIN_AUTH_REQUIRED",
				"Admin authentication required",
			),
		)

		return
	}

	result, err :=
		h.service.SetCustomerStatus(
			c.Request.Context(),
			customerID,
			status,
			metadata,
		)

	switch {
	case errors.Is(
		err,
		ErrAdminCustomerNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"CUSTOMER_NOT_FOUND",
				"Customer not found",
			),
		)

		return

	case errors.Is(
		err,
		ErrInvalidAdminCustomerStatus,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_CUSTOMER_STATUS",
				"Invalid customer status",
			),
		)

		return

	case err != nil:
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
