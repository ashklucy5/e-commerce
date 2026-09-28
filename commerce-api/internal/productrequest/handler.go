package productrequest

import (
	"errors"
	"net/http"
	"strconv"

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

func (h *Handler) Create(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	var request CreateRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.Create(
			c.Request.Context(),
			customerID,
			request,
		)
	if err != nil {
		writeError(
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

func (h *Handler) List(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	limit, err :=
		queryInt(
			c.Query(
				"limit",
			),
		)
	if err != nil {
		writeError(
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
		writeError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.List(
			c.Request.Context(),
			customerID,
			limit,
			offset,
		)
	if err != nil {
		writeError(
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

func (h *Handler) Get(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	result, err :=
		h.service.Get(
			c.Request.Context(),
			customerID,
			c.Param(
				"id",
			),
		)
	if err != nil {
		writeError(
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

func (h *Handler) ListMessages(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	limit, err :=
		queryInt(
			c.Query(
				"limit",
			),
		)
	if err != nil {
		writeError(
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
		writeError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.
			ListMessages(
				c.Request.Context(),
				customerID,
				c.Param(
					"id",
				),
				limit,
				offset,
			)
	if err != nil {
		writeError(
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

func (h *Handler) AddMessage(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	var request AddMessageRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	result, err :=
		h.service.
			AddCustomerMessage(
				c.Request.Context(),
				customerID,
				c.Param(
					"id",
				),
				request,
			)
	if err != nil {
		writeError(
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

func queryInt(
	value string,
) (
	int,
	error,
) {
	if value == "" {
		return 0,
			nil
	}

	return strconv.Atoi(
		value,
	)
}

func writeError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		auth.ErrInvalidAccessToken,
	):
		writeJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

	case errors.Is(
		err,
		ErrInvalidRequest,
	):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_PRODUCT_REQUEST",
			"Invalid product request",
		)

	case errors.Is(
		err,
		ErrNotFound,
	):
		writeJSONError(
			c,
			http.StatusNotFound,
			"PRODUCT_REQUEST_NOT_FOUND",
			"Product request was not found",
		)

	case errors.Is(
		err,
		ErrConversationClosed,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"PRODUCT_REQUEST_CONVERSATION_CLOSED",
			"This product request can no longer receive customer messages",
		)

	default:
		writeJSONError(
			c,
			http.StatusInternalServerError,
			"PRODUCT_REQUEST_FAILED",
			"Unable to process product request",
		)
	}
}

func writeJSONError(
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
