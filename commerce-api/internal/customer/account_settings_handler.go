package customer

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
)

func (h *Handler) GetPreferences(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.GetPreferences(
			c.Request.Context(),
			customerID,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) UpdatePreferences(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	var request UpdatePreferencesRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid customer preferences request",
		)

		return
	}

	result, err :=
		h.service.UpdatePreferences(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) GetNotificationPreferences(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.GetNotificationPreferences(
			c.Request.Context(),
			customerID,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) UpdateNotificationPreferences(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	var request UpdateNotificationPreferencesRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid notification preferences request",
		)

		return
	}

	result, err :=
		h.service.UpdateNotificationPreferences(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) GetPrivacySettings(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.GetPrivacySettings(
			c.Request.Context(),
			customerID,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) UpdatePrivacySettings(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	var request UpdatePrivacySettingsRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid privacy settings request",
		)

		return
	}

	result, err :=
		h.service.UpdatePrivacySettings(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) GetStyleProfile(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.GetStyleProfile(
			c.Request.Context(),
			customerID,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) PutStyleProfile(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	var request PutStyleProfileRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid style profile request",
		)

		return
	}

	result, err :=
		h.service.PutStyleProfile(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) ListSavedSizes(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.ListSavedSizes(
			c.Request.Context(),
			customerID,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) CreateSavedSize(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	var request SaveSizeRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid saved size request",
		)

		return
	}

	result, err :=
		h.service.CreateSavedSize(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) ReplaceSavedSize(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	var request SaveSizeRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid saved size request",
		)

		return
	}

	result, err :=
		h.service.ReplaceSavedSize(
			c.Request.Context(),
			customerID,
			c.Param(
				"size_id",
			),
			request,
		)
	if err != nil {
		writeAccountSettingsError(
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

func (h *Handler) DeleteSavedSize(
	c *gin.Context,
) {
	customerID, ok :=
		accountCustomerID(
			c,
		)
	if !ok {
		return
	}

	if err :=
		h.service.DeleteSavedSize(
			c.Request.Context(),
			customerID,
			c.Param(
				"size_id",
			),
		); err != nil {
		writeAccountSettingsError(
			c,
			err,
		)

		return
	}

	c.Status(
		http.StatusNoContent,
	)
}

func accountCustomerID(
	c *gin.Context,
) (string, bool) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if ok {
		return customerID, true
	}

	writeCustomerJSONError(
		c,
		http.StatusUnauthorized,
		"UNAUTHORIZED",
		"Authentication required",
	)

	return "", false
}

func writeAccountSettingsError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrNoChanges,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"NO_CHANGES",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidPreferences,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_PREFERENCES",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidStyleProfile,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_STYLE_PROFILE",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidSavedSize,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SAVED_SIZE",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidSavedSizeID,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SAVED_SIZE_ID",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrSavedSizeExists,
	):
		writeCustomerJSONError(
			c,
			http.StatusConflict,
			"SAVED_SIZE_ALREADY_EXISTS",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrSavedSizeNotFound,
	):
		writeCustomerJSONError(
			c,
			http.StatusNotFound,
			"SAVED_SIZE_NOT_FOUND",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrCustomerNotFound,
	):
		writeCustomerJSONError(
			c,
			http.StatusNotFound,
			"CUSTOMER_NOT_FOUND",
			"Customer not found",
		)

	default:
		writeCustomerJSONError(
			c,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Unable to process customer account settings request",
		)
	}
}
