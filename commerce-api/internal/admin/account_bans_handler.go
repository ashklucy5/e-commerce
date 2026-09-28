package admin

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

type createStaffBanRequest struct {
	BanType string `json:"ban_type"`
	Reason  string `json:"reason"`

	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type revokeStaffBanRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) StaffBans(
	c *gin.Context,
) {
	result, err :=
		h.service.ListStaffBans(
			c.Request.Context(),
			strings.TrimSpace(
				c.Param(
					"staff_id",
				),
			),
		)
	if err != nil {
		writeAdminAccountBanError(
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

func (h *Handler) CreateStaffBan(
	c *gin.Context,
) {
	metadata, ok :=
		accountBanActionMetadata(
			c,
		)
	if !ok {
		return
	}

	var request createStaffBanRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminAccountBanError(
			c,
			ErrAdminBanInvalidInput,
		)

		return
	}

	result, err :=
		h.service.CreateStaffBan(
			c.Request.Context(),
			strings.TrimSpace(
				c.Param(
					"staff_id",
				),
			),
			CreateAccountBanInput{
				Scope: AccountBanScopeFullAccount,

				BanType: request.BanType,

				Reason: request.Reason,

				ExpiresAt: request.ExpiresAt,
			},
			metadata,
		)
	if err != nil {
		writeAdminAccountBanError(
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

func (h *Handler) RevokeStaffBan(
	c *gin.Context,
) {
	metadata, ok :=
		accountBanActionMetadata(
			c,
		)
	if !ok {
		return
	}

	var request revokeStaffBanRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminAccountBanError(
			c,
			ErrAdminBanInvalidInput,
		)

		return
	}

	result, err :=
		h.service.RevokeStaffBan(
			c.Request.Context(),
			strings.TrimSpace(
				c.Param(
					"staff_id",
				),
			),
			strings.TrimSpace(
				c.Param(
					"ban_id",
				),
			),
			request.Reason,
			metadata,
		)
	if err != nil {
		writeAdminAccountBanError(
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

func accountBanActionMetadata(
	c *gin.Context,
) (
	AdminActionMetadata,
	bool,
) {
	metadata, ok :=
		adminActionMetadataFromContext(
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

		return AdminActionMetadata{},
			false
	}

	return metadata, true
}

func writeAdminAccountBanError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrAdminBanInvalidInput,
	):
		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_BAN",
				"Invalid staff ban request",
			),
		)

	case errors.Is(
		err,
		ErrAdminBanTargetNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"STAFF_ACCOUNT_NOT_FOUND",
				"The requested staff account was not found",
			),
		)

	case errors.Is(
		err,
		ErrAdminBanNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"STAFF_BAN_NOT_FOUND",
				"The requested staff ban was not found",
			),
		)

	case errors.Is(
		err,
		ErrAdminTemporaryFullAccountBanUnsupported,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"TEMPORARY_STAFF_BAN_NOT_AVAILABLE",
				"Temporary staff bans are not available until automatic expiry restoration is enabled",
			),
		)

	case errors.Is(
		err,
		ErrAdminBanConflict,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_BAN_ALREADY_ACTIVE",
				"An active staff ban already exists",
			),
		)

	case errors.Is(
		err,
		ErrAdminBanAlreadyRevoked,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_BAN_ALREADY_REVOKED",
				"This staff ban has already been revoked",
			),
		)

	case errors.Is(
		err,
		ErrAdminBanNotActive,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_BAN_NOT_ACTIVE",
				"This staff ban is no longer active",
			),
		)

	case errors.Is(
		err,
		ErrAdminBanStateConflict,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_BAN_STATE_CONFLICT",
				"The staff account state no longer matches the ban state",
			),
		)

	case errors.Is(
		err,
		ErrAdminSelfBan,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"SELF_BAN_NOT_ALLOWED",
				"You cannot ban or unban your own staff account",
			),
		)

	case errors.Is(
		err,
		ErrAdminProtectedStaffMutation,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"PROTECTED_STAFF_REQUIRES_SUPER_ADMIN",
				"Only a Super Admin can change ban state for protected staff",
			),
		)

	case errors.Is(
		err,
		ErrAdminLastSuperAdmin,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"LAST_SUPER_ADMIN_REQUIRED",
				"The last active Super Admin cannot be banned",
			),
		)

	case errors.Is(
		err,
		ErrAdminStaffDeleted,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_ACCOUNT_DELETED",
				"Deleted staff accounts cannot be changed",
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
