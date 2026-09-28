package staff

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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

func (h *Handler) Login(
	c *gin.Context,
) {
	var request LoginRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeStaffError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.Login(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeStaffError(
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

func (h *Handler) Refresh(
	c *gin.Context,
) {
	var request RefreshRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeStaffError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.Refresh(
			c.Request.Context(),
			request,
		)
	if err != nil {
		writeStaffError(
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

func (h *Handler) Logout(
	c *gin.Context,
) {
	token, err :=
		extractBearerToken(
			c.GetHeader(
				"Authorization",
			),
		)
	if err != nil {
		writeStaffError(
			c,
			err,
		)

		return
	}

	if err :=
		h.service.Logout(
			c.Request.Context(),
			token,
		); err != nil {
		writeStaffError(
			c,
			err,
		)

		return
	}

	c.Status(
		http.StatusNoContent,
	)
}

func (h *Handler) Me(
	c *gin.Context,
) {
	account, ok :=
		AccountFromContext(
			c,
		)

	if !ok {
		writeStaffError(
			c,
			ErrInvalidAccessToken,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": account,
		},
	)
}

func writeStaffError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidRequest,
	):
		writeStaffJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid staff authentication request",
		)

	case errors.Is(
		err,
		ErrInvalidCredentials,
	):
		writeStaffJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_STAFF_CREDENTIALS",
			"Invalid staff credentials",
		)

	case errors.Is(
		err,
		ErrStaffDisabled,
	):
		writeStaffJSONError(
			c,
			http.StatusForbidden,
			"STAFF_ACCOUNT_DISABLED",
			"Staff account is not active",
		)

	case errors.Is(
		err,
		ErrInvalidAccessToken,
	):
		writeStaffJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_STAFF_ACCESS_TOKEN",
			"Invalid or expired staff access token",
		)

	case errors.Is(
		err,
		ErrInvalidRefreshToken,
	):
		writeStaffJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_STAFF_REFRESH_TOKEN",
			"Invalid or expired staff refresh token",
		)

	case errors.Is(
		err,
		ErrForbidden,
	):
		writeStaffJSONError(
			c,
			http.StatusForbidden,
			"STAFF_FORBIDDEN",
			"Staff account does not have permission",
		)

	default:
		writeStaffJSONError(
			c,
			http.StatusInternalServerError,
			"STAFF_AUTH_FAILED",
			"Unable to process staff authentication",
		)
	}
}

func writeStaffJSONError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(
		status,
		gin.H{
			"error": gin.H{
				"code": code,

				"message": message,
			},
		},
	)
}
