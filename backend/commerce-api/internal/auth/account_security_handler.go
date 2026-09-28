package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetCustomerSecurityOverview(
	c *gin.Context,
) {
	customerID, ok :=
		CustomerIDFromContext(
			c,
		)
	if !ok {
		writeAuthJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

		return
	}

	accessToken, err :=
		extractBearerToken(
			c.GetHeader(
				"Authorization",
			),
		)
	if err != nil {
		writeCustomerSecurityError(
			c,
			err,
		)

		return
	}

	result, err :=
		h.service.GetCustomerSecurityOverview(
			c.Request.Context(),
			customerID,
			accessToken,
		)
	if err != nil {
		writeCustomerSecurityError(
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

func (h *Handler) ListCustomerSessions(
	c *gin.Context,
) {
	customerID, ok :=
		CustomerIDFromContext(
			c,
		)
	if !ok {
		writeAuthJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

		return
	}

	accessToken, err :=
		extractBearerToken(
			c.GetHeader(
				"Authorization",
			),
		)
	if err != nil {
		writeCustomerSecurityError(
			c,
			err,
		)

		return
	}

	result, err :=
		h.service.ListCustomerSessions(
			c.Request.Context(),
			customerID,
			accessToken,
		)
	if err != nil {
		writeCustomerSecurityError(
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

func (h *Handler) RevokeCustomerSession(
	c *gin.Context,
) {
	customerID, ok :=
		CustomerIDFromContext(
			c,
		)
	if !ok {
		writeAuthJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

		return
	}

	if err :=
		h.service.RevokeCustomerSession(
			c.Request.Context(),
			customerID,
			c.Param(
				"session_id",
			),
		); err != nil {

		writeCustomerSecurityError(
			c,
			err,
		)

		return
	}

	c.Status(
		http.StatusNoContent,
	)
}

func (h *Handler) RevokeOtherCustomerSessions(
	c *gin.Context,
) {
	customerID, ok :=
		CustomerIDFromContext(
			c,
		)
	if !ok {
		writeAuthJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

		return
	}

	accessToken, err :=
		extractBearerToken(
			c.GetHeader(
				"Authorization",
			),
		)
	if err != nil {
		writeCustomerSecurityError(
			c,
			err,
		)

		return
	}

	result, err :=
		h.service.RevokeOtherCustomerSessions(
			c.Request.Context(),
			customerID,
			accessToken,
		)
	if err != nil {
		writeCustomerSecurityError(
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

func writeCustomerSecurityError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidAccessToken,
	):
		writeAuthJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

	case errors.Is(
		err,
		ErrInvalidCustomerSessionID,
	):
		writeAuthJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SESSION_ID",
			"Invalid customer session id",
		)

	case errors.Is(
		err,
		ErrCustomerSessionNotFound,
	):
		writeAuthJSONError(
			c,
			http.StatusNotFound,
			"SESSION_NOT_FOUND",
			"Customer session not found",
		)

	default:
		writeAuthJSONError(
			c,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Unable to process customer security request",
		)
	}
}
