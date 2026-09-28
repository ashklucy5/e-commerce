package admin

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformpagination "project.local/commerce-api/internal/platform/pagination"
	platformvalidation "project.local/commerce-api/internal/platform/validation"
)

type Handler struct {
	service *Service
}

func NewHandler(
	service *Service,
) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Dashboard(
	c *gin.Context,
) {
	result, err :=
		h.service.Dashboard(
			c.Request.Context(),
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
			"data": result,
		},
	)
}

func (h *Handler) AuditEvents(
	c *gin.Context,
) {
	params, err :=
		platformpagination.Parse(
			c.Query(
				"page",
			),
			c.Query(
				"limit",
			),
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PAGINATION",
				"Invalid pagination",
			),
		)

		return
	}

	staffID :=
		strings.TrimSpace(
			c.Query(
				"staff_id",
			),
		)

	if staffID != "" &&
		!platformvalidation.IsUUID(
			staffID,
		) {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_ID",
				"staff_id must be a valid UUID",
			),
		)

		return
	}

	outcome :=
		strings.ToLower(
			strings.TrimSpace(
				c.Query(
					"outcome",
				),
			),
		)

	if outcome != "" &&
		outcome != "success" &&
		outcome != "failure" &&
		outcome != "blocked" &&
		outcome != "info" {

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_AUDIT_OUTCOME",
				"Invalid audit outcome",
			),
		)

		return
	}

	result, err :=
		h.service.AuditEvents(
			c.Request.Context(),
			params,
			AuditFilter{
				EventType: c.Query(
					"event_type",
				),

				Outcome: outcome,

				StaffID: staffID,
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
