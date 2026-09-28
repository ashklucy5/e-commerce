package customer

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/platform/storage"
)

func (h *Handler) CreateAvatarUploadTarget(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeCustomerJSONError(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication required",
		)

		return
	}

	var request AvatarUploadTargetRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid avatar upload request",
		)

		return
	}

	result, err :=
		h.service.CreateAvatarUploadTarget(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeAvatarError(
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

func (h *Handler) CompleteAvatarUpload(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeCustomerJSONError(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication required",
		)

		return
	}

	var request AvatarCompleteRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid avatar completion request",
		)

		return
	}

	result, err :=
		h.service.CompleteAvatarUpload(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeAvatarError(
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

func (h *Handler) DeleteAvatar(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeCustomerJSONError(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication required",
		)

		return
	}

	if err :=
		h.service.DeleteAvatar(
			c.Request.Context(),
			customerID,
		); err != nil {
		writeAvatarError(
			c,
			err,
		)

		return
	}

	c.Status(
		http.StatusNoContent,
	)
}

func writeAvatarError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidAvatarContentType,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_AVATAR_CONTENT_TYPE",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidAvatarSize,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_AVATAR_SIZE",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidAvatarKey,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_AVATAR_KEY",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrAvatarUploadNotFound,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"AVATAR_UPLOAD_NOT_FOUND",
			err.Error(),
		)

	case errors.Is(
		err,
		storage.ErrDisabled,
	):
		writeCustomerJSONError(
			c,
			http.StatusServiceUnavailable,
			"STORAGE_DISABLED",
			"Object storage is unavailable",
		)

	default:
		writeCustomerError(
			c,
			err,
		)
	}
}
