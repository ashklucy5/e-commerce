package adminauth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func (h *Handler) SetActivationPassword(
	c *gin.Context,
) {
	var request StaffActivationPasswordRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeStaffActivationError(
			c,
			ErrInvalidStaffActivation,
		)

		return
	}

	result, err :=
		h.service.SetActivationPassword(
			c.Request.Context(),
			request,
			metadataFromGin(
				c,
			),
		)
	if err != nil {
		writeStaffActivationError(
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

func (h *Handler) BeginActivationMFA(
	c *gin.Context,
) {
	var request StaffActivationMFAEnrollRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeStaffActivationError(
			c,
			ErrInvalidStaffActivation,
		)

		return
	}

	result, err :=
		h.service.BeginActivationMFA(
			c.Request.Context(),
			request,
			metadataFromGin(
				c,
			),
		)
	if err != nil {
		writeStaffActivationError(
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

func (h *Handler) ConfirmActivationMFA(
	c *gin.Context,
) {
	var request StaffActivationMFAConfirmRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeStaffActivationError(
			c,
			ErrInvalidStaffActivation,
		)

		return
	}

	result, err :=
		h.service.ConfirmActivationMFA(
			c.Request.Context(),
			request,
			metadataFromGin(
				c,
			),
		)
	if err != nil {
		writeStaffActivationError(
			c,
			err,
		)

		return
	}

	/*
		Do not create/set Admin session cookies here.

		Activation establishes the staff credential state only.
		The staff member must subsequently authenticate through
		the normal /auth/login flow.
	*/
	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func writeStaffActivationError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidStaffActivation,
	):
		platformerrors.Write(
			c,
			platformerrors.Unauthorized(
				"INVALID_STAFF_ACTIVATION",
				"Invalid staff activation credentials",
			),
		)

	case errors.Is(
		err,
		ErrStaffActivationExpired,
	):
		platformerrors.Write(
			c,
			platformerrors.Unauthorized(
				"STAFF_ACTIVATION_EXPIRED",
				"This staff activation has expired",
			),
		)

	case errors.Is(
		err,
		ErrStaffActivationCompleted,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_ACTIVATION_COMPLETED",
				"This staff account has already completed activation",
			),
		)

	case errors.Is(
		err,
		ErrStaffActivationPasswordAlreadySet,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_ACTIVATION_PASSWORD_ALREADY_SET",
				"The staff password has already been set; continue with MFA enrollment",
			),
		)

	case errors.Is(
		err,
		ErrStaffActivationPasswordNotSet,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_ACTIVATION_PASSWORD_REQUIRED",
				"Set the staff password before enrolling MFA",
			),
		)

	case errors.Is(
		err,
		ErrStaffActivationStateConflict,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"STAFF_ACTIVATION_STATE_CONFLICT",
				"This staff account cannot currently be activated",
			),
		)

	case errors.Is(
		err,
		ErrAdminPanelAccessRequired,
	):
		platformerrors.Write(
			c,
			platformerrors.Forbidden(
				"ADMIN_PANEL_ACCESS_REQUIRED",
				"This staff account does not have Admin panel access",
			),
		)

	case errors.Is(
		err,
		ErrPasswordRequired,
	),
		errors.Is(
			err,
			ErrPasswordTooShort,
		),
		errors.Is(
			err,
			ErrPasswordTooLong,
		),
		errors.Is(
			err,
			ErrPasswordCommon,
		):

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_STAFF_PASSWORD",
				err.Error(),
			),
		)

	default:
		/*
			Reuse the established Admin-auth mapping for MFA
			lockouts, invalid codes, enrollment state,
			configuration failures and infrastructure failures.
		*/
		writeAdminAuthError(
			c,
			err,
		)
	}
}
