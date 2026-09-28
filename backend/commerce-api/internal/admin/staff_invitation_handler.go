package admin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func (h *Handler) StaffInvitation(
	c *gin.Context,
) {
	staffID, ok := adminStaffIDParam(c)
	if !ok {
		return
	}

	result, err := h.service.GetStaffInvitation(
		c.Request.Context(),
		staffID,
	)
	if writeAdminStaffInvitationError(c, err) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{"data": result},
	)
}

func (h *Handler) ReissueStaffInvitation(
	c *gin.Context,
) {
	staffID, ok := adminStaffIDParam(c)
	if !ok {
		return
	}

	metadata, ok := adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err := h.service.ReissueStaffInvitation(
		c.Request.Context(),
		staffID,
		metadata,
	)
	if writeAdminStaffInvitationError(c, err) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{"data": result},
	)
}

func (h *Handler) CancelStaffInvitation(
	c *gin.Context,
) {
	staffID, ok := adminStaffIDParam(c)
	if !ok {
		return
	}

	metadata, ok := adminMutationMetadata(c)
	if !ok {
		return
	}

	result, err := h.service.CancelStaffInvitation(
		c.Request.Context(),
		staffID,
		metadata,
	)
	if writeAdminStaffInvitationError(c, err) {
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{"data": result},
	)
}

func writeAdminStaffInvitationError(
	c *gin.Context,
	err error,
) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, ErrAdminStaffInvitationNotFound):
		platformerrors.Write(
			c,
			platformerrors.NotFound(
				"STAFF_INVITATION_NOT_FOUND",
				"Staff invitation not found",
			),
		)

	case errors.Is(err, ErrAdminStaffInvitationState):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_INVITATION_STATE_CONFLICT",
				"Invitations can only be managed while the staff account is pending activation",
			),
		)

	case errors.Is(err, ErrAdminStaffInvitationNotPending):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_INVITATION_NOT_PENDING",
				"There is no pending staff invitation to cancel",
			),
		)

	case errors.Is(err, ErrAdminStaffInvitationMFAConflict):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_INVITATION_MFA_STATE_CONFLICT",
				"The pending staff account already has active Admin MFA",
			),
		)

	default:
		return writeAdminStaffRoleError(c, err)
	}

	return true
}
