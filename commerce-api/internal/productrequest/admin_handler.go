package productrequest

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/adminauth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
)

type AdminHandler struct {
	service *AdminService
}

func NewAdminHandler(
	service *AdminService,
) *AdminHandler {
	return &AdminHandler{
		service: service,
	}
}

func (h *AdminHandler) List(
	c *gin.Context,
) {
	limit, err :=
		queryInt(
			c.Query(
				"limit",
			),
		)
	if err != nil {
		writeAdminError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	offset, err :=
		queryInt(
			c.Query(
				"offset",
			),
		)
	if err != nil {
		writeAdminError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.List(
			c.Request.Context(),
			AdminListFilter{
				Status: c.Query(
					"status",
				),

				Query: c.Query(
					"q",
				),

				Limit: limit,

				Offset: offset,
			},
		)
	if err != nil {
		writeAdminError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result.Items,

			"meta": gin.H{
				"limit": result.Limit,

				"offset": result.Offset,
			},
		},
	)
}

func (h *AdminHandler) Get(
	c *gin.Context,
) {
	result, err :=
		h.service.Get(
			c.Request.Context(),
			strings.TrimSpace(
				c.Param(
					"id",
				),
			),
		)
	if err != nil {
		writeAdminError(
			c,
			err,
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

func (h *AdminHandler) ListMessages(
	c *gin.Context,
) {
	limit, err :=
		queryInt(
			c.Query(
				"limit",
			),
		)
	if err != nil {
		writeAdminError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	offset, err :=
		queryInt(
			c.Query(
				"offset",
			),
		)
	if err != nil {
		writeAdminError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.ListMessages(
			c.Request.Context(),
			strings.TrimSpace(
				c.Param(
					"id",
				),
			),
			limit,
			offset,
		)
	if err != nil {
		writeAdminError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result.Items,

			"meta": gin.H{
				"limit": result.Limit,

				"offset": result.Offset,
			},
		},
	)
}

func (h *AdminHandler) AddMessage(
	c *gin.Context,
) {
	staffAccountID, ok :=
		adminStaffAccountID(
			c,
		)
	if !ok {
		return
	}

	var request AdminAddMessageRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.AddMessage(
			c.Request.Context(),
			staffAccountID,
			strings.TrimSpace(
				c.Param(
					"id",
				),
			),
			request,
		)
	if err != nil {
		writeAdminError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": result,
		},
	)
}

func (h *AdminHandler) UpdateStatus(
	c *gin.Context,
) {
	staffAccountID, ok :=
		adminStaffAccountID(
			c,
		)
	if !ok {
		return
	}

	var request AdminUpdateStatusRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.UpdateStatus(
			c.Request.Context(),
			staffAccountID,
			strings.TrimSpace(
				c.Param(
					"id",
				),
			),
			request,
		)
	if err != nil {
		writeAdminError(
			c,
			err,
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

func adminStaffAccountID(
	c *gin.Context,
) (
	string,
	bool,
) {
	principal, ok :=
		adminauth.PrincipalFromContext(
			c,
		)

	if !ok ||
		strings.TrimSpace(
			principal.Staff.ID,
		) == "" {

		platformerrors.Abort(
			c,
			platformerrors.Unauthorized(
				"ADMIN_AUTHENTICATION_REQUIRED",
				"Admin authentication required",
			),
		)

		return "",
			false
	}

	return strings.TrimSpace(
			principal.Staff.ID,
		),
		true
}

func writeAdminError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidRequest,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_PRODUCT_REQUEST",
				"Invalid product request input",
			),
		)

	case errors.Is(
		err,
		ErrNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"PRODUCT_REQUEST_NOT_FOUND",
				"Product request not found",
			),
		)

	case errors.Is(
		err,
		ErrInvalidStatusTransition,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"INVALID_PRODUCT_REQUEST_STATUS_TRANSITION",
				"Product request cannot move to the requested status from its current state",
			),
		)

	case errors.Is(
		err,
		ErrConversationClosed,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"PRODUCT_REQUEST_CONVERSATION_CLOSED",
				"This product request conversation is closed",
			),
		)

	case errors.Is(
		err,
		ErrAdminActorUnavailable,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"ADMIN_SUPPORT_ACTOR_UNAVAILABLE",
				"The authenticated staff account cannot act as a support actor",
			),
		)

	default:
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)
	}
}
