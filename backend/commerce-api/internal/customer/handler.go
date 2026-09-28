package customer

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
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

func (h *Handler) GetMe(
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

	result, err :=
		h.service.GetProfile(
			c.Request.Context(),
			customerID,
		)
	if err != nil {
		writeCustomerError(
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

func (h *Handler) UpdateMe(
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

	var request UpdateProfileRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid customer profile request",
		)

		return
	}

	result, err :=
		h.service.UpdateProfile(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeCustomerError(
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

func (h *Handler) ListAddresses(
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

	result, err :=
		h.service.ListAddresses(
			c.Request.Context(),
			customerID,
		)
	if err != nil {
		writeCustomerError(
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

func (h *Handler) CreateAddress(
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

	var request CreateAddressRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid customer address request",
		)

		return
	}

	result, err :=
		h.service.CreateAddress(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeCustomerError(
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

func (h *Handler) UpdateAddress(
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

	var request UpdateAddressRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"Invalid customer address request",
		)

		return
	}

	result, err :=
		h.service.UpdateAddress(
			c.Request.Context(),
			customerID,
			c.Param(
				"address_id",
			),
			request,
		)
	if err != nil {
		writeCustomerError(
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

func (h *Handler) DeleteAddress(
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

	err :=
		h.service.DeleteAddress(
			c.Request.Context(),
			customerID,
			c.Param(
				"address_id",
			),
		)
	if err != nil {
		writeCustomerError(
			c,
			err,
		)

		return
	}

	c.Status(
		http.StatusNoContent,
	)
}

func writeCustomerError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidPhone,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_PHONE",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidEmail,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_EMAIL",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidFullName,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_FULL_NAME",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrInvalidAddress,
	):
		writeCustomerJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_ADDRESS",
			err.Error(),
		)

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
		ErrPhoneInUse,
	):
		writeCustomerJSONError(
			c,
			http.StatusConflict,
			"PHONE_ALREADY_REGISTERED",
			err.Error(),
		)

	case errors.Is(
		err,
		ErrEmailInUse,
	):
		writeCustomerJSONError(
			c,
			http.StatusConflict,
			"EMAIL_ALREADY_REGISTERED",
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

	case errors.Is(
		err,
		ErrAddressNotFound,
	):
		writeCustomerJSONError(
			c,
			http.StatusNotFound,
			"ADDRESS_NOT_FOUND",
			"Customer address not found",
		)

	default:
		writeCustomerJSONError(
			c,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Unable to process customer request",
		)
	}
}

func writeCustomerJSONError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(
		status,
		gin.H{
			"error": gin.H{
				"code":    code,
				"message": message,
			},
		},
	)
}
