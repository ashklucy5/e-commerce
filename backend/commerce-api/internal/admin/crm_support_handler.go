package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/adminauth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
	"project.local/commerce-api/internal/support"
)

func (h *Handler) CRMQueues(c *gin.Context) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(c)
	if !ok {
		return
	}

	result, err :=
		support.NewAdminService(
			h.service.db,
		).ListQueues(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
		)
	if err != nil {
		writeAdminCRMError(
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

func (h *Handler) CRMCases(c *gin.Context) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(c)
	if !ok {
		return
	}

	limit, err :=
		adminCRMQueryInt(
			c.Query(
				"limit",
			),
		)
	if err != nil {
		writeAdminCRMError(
			c,
			support.ErrCaseNotFound,
		)
		return
	}

	offset, err :=
		adminCRMQueryInt(
			c.Query(
				"offset",
			),
		)
	if err != nil {
		writeAdminCRMError(
			c,
			support.ErrCaseNotFound,
		)
		return
	}

	items,
		resolvedLimit,
		resolvedOffset,
		err :=
		support.NewAdminService(
			h.service.db,
		).ListCases(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
			limit,
			offset,
		)
	if err != nil {
		writeAdminCRMError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": items,
			"meta": gin.H{
				"limit":  resolvedLimit,
				"offset": resolvedOffset,
			},
		},
	)
}

func (h *Handler) CRMCase(c *gin.Context) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(c)
	if !ok {
		return
	}

	result, err :=
		support.NewAdminService(
			h.service.db,
		).GetCase(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
			c.Param(
				"id",
			),
		)
	if err != nil {
		writeAdminCRMError(
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

func (h *Handler) CRMCaseMessages(c *gin.Context) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(c)
	if !ok {
		return
	}

	limit, err :=
		adminCRMQueryInt(
			c.Query(
				"limit",
			),
		)
	if err != nil {
		writeAdminCRMError(
			c,
			support.ErrCaseNotFound,
		)
		return
	}

	offset, err :=
		adminCRMQueryInt(
			c.Query(
				"offset",
			),
		)
	if err != nil {
		writeAdminCRMError(
			c,
			support.ErrCaseNotFound,
		)
		return
	}

	items,
		resolvedLimit,
		resolvedOffset,
		err :=
		support.NewAdminService(
			h.service.db,
		).ListMessages(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
			c.Param(
				"id",
			),
			limit,
			offset,
		)
	if err != nil {
		writeAdminCRMError(
			c,
			err,
		)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": items,
			"meta": gin.H{
				"limit":  resolvedLimit,
				"offset": resolvedOffset,
			},
		},
	)
}

func (h *Handler) ClaimCRMCase(c *gin.Context) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(c)
	if !ok {
		return
	}

	result, err :=
		support.NewAdminService(
			h.service.db,
		).ClaimCase(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
			c.Param(
				"id",
			),
		)
	if err != nil {
		writeAdminCRMError(
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

func (h *Handler) ReplyCRMCase(c *gin.Context) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(c)
	if !ok {
		return
	}

	var request support.AdminReplyWithAttachmentsRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminCRMError(
			c,
			support.ErrInvalidMessage,
		)
		return
	}

	result, err :=
		support.NewAdminService(
			h.service.db,
		).ReplyWithAttachments(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
			c.Param(
				"id",
			),
			request,
		)
	if err != nil {
		writeAdminCRMError(
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

func (h *Handler) ResolveCRMCase(c *gin.Context) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(c)
	if !ok {
		return
	}

	result, err :=
		support.NewAdminService(
			h.service.db,
		).Resolve(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
			c.Param(
				"id",
			),
		)
	if err != nil {
		writeAdminCRMError(
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

func (h *Handler) CloseCRMCase(
	c *gin.Context,
) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		support.NewAdminService(
			h.service.db,
		).Close(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
			c.Param(
				"id",
			),
		)
	if err != nil {
		writeAdminCRMError(
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

func (h *Handler) EscalateCRMCase(c *gin.Context) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(c)
	if !ok {
		return
	}

	var request support.EscalateRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminCRMError(
			c,
			support.ErrInvalidQueue,
		)
		return
	}

	result, err :=
		support.NewAdminService(
			h.service.db,
		).Escalate(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
			c.Param(
				"id",
			),
			request,
		)
	if err != nil {
		writeAdminCRMError(
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

func adminCRMPrincipal(
	c *gin.Context,
) (
	adminauth.AdminPrincipal,
	bool,
	bool,
) {
	principal, ok :=
		adminauth.PrincipalFromContext(
			c,
		)
	if !ok {
		platformerrors.Write(
			c,
			platformerrors.Unauthorized(
				"ADMIN_AUTHENTICATION_REQUIRED",
				"Admin authentication required",
			),
		)

		return adminauth.AdminPrincipal{},
			false,
			false
	}

	queueScoped :=
		false

	hasProtectedAdminRole :=
		false

	for _, role := range principal.Staff.Roles {
		switch role {
		case RoleSuperAdmin,
			RoleAdministrator:

			hasProtectedAdminRole =
				true

		case support.RoleAgent,
			support.RoleSupervisor:

			queueScoped =
				true
		}
	}

	if hasProtectedAdminRole {
		queueScoped =
			false
	}

	return principal,
		queueScoped,
		true
}

func adminCRMQueryInt(
	value string,
) (
	int,
	error,
) {
	if value == "" {
		return 0, nil
	}

	return strconv.Atoi(
		value,
	)
}

func writeAdminCRMError(
	c *gin.Context,
	err error,
) {
	var apiErr error

	switch {
	case errors.Is(
		err,
		support.ErrNotSupportActor,
	):
		apiErr =
			platformerrors.Forbidden(
				"ADMIN_CRM_SUPPORT_ACTOR_REQUIRED",
				"This staff account is not configured for support operations",
			)

	case errors.Is(
		err,
		support.ErrSupportActorDisabled,
	):
		apiErr =
			platformerrors.Forbidden(
				"ADMIN_CRM_SUPPORT_ACTOR_DISABLED",
				"Support operations are disabled for this staff account",
			)

	case errors.Is(
		err,
		support.ErrCaseNotFound,
	):
		apiErr =
			platformerrors.NotFound(
				"ADMIN_CRM_CASE_NOT_FOUND",
				"CRM case was not found",
			)

	case errors.Is(
		err,
		support.ErrCaseAlreadyClaimed,
	):
		apiErr =
			platformerrors.Conflict(
				"ADMIN_CRM_CASE_ALREADY_CLAIMED",
				"CRM case is already claimed by another support actor",
			)

	case errors.Is(
		err,
		support.ErrCaseNotOwned,
	):
		apiErr =
			platformerrors.Conflict(
				"ADMIN_CRM_CASE_NOT_OWNED",
				"Claim the CRM case before performing this action",
			)

	case errors.Is(
		err,
		support.ErrCaseNotResolved,
	):
		apiErr =
			platformerrors.Conflict(
				"ADMIN_CRM_CASE_NOT_RESOLVED",
				"Resolve the CRM case before closing it",
			)

	case errors.Is(
		err,
		support.ErrCaseClosed,
	):
		apiErr =
			platformerrors.Conflict(
				"ADMIN_CRM_CASE_CLOSED",
				"Closed CRM cases cannot be modified",
			)

	case errors.Is(
		err,
		support.ErrActorCapacity,
	):
		apiErr =
			platformerrors.Conflict(
				"ADMIN_CRM_CAPACITY_REACHED",
				"The support actor has reached the active case limit",
			)

	case errors.Is(
		err,
		support.ErrInvalidMessage,
	):
		apiErr =
			platformerrors.BadRequest(
				"INVALID_ADMIN_CRM_MESSAGE",
				"CRM message is invalid",
			)

	case errors.Is(
		err,
		support.ErrInvalidVisibility,
	):
		apiErr =
			platformerrors.BadRequest(
				"INVALID_ADMIN_CRM_VISIBILITY",
				"Visibility must be customer or internal",
			)

	case errors.Is(
		err,
		support.ErrInvalidQueue,
	):
		apiErr =
			platformerrors.BadRequest(
				"INVALID_ADMIN_CRM_QUEUE",
				"CRM support queue is invalid",
			)

	default:
		apiErr =
			platformerrors.Internal(
				err,
			)
	}

	platformerrors.Write(
		c,
		apiErr,
	)
}
