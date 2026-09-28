package returns

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

func (h *Handler) Create(
	c *gin.Context,
) {
	var request CreateRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeError(
			c,
			ErrInvalidInput,
		)

		return
	}

	result, err :=
		h.service.Create(
			c.Request.Context(),
			c.Param(
				"order_id",
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

func (h *Handler) Get(
	c *gin.Context,
) {
	result, err :=
		h.service.Get(
			c.Request.Context(),
			c.Param(
				"return_id",
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

func (h *Handler) Timeline(
	c *gin.Context,
) {
	result, err :=
		h.service.Timeline(
			c.Request.Context(),
			c.Param(
				"return_id",
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

func (h *Handler) Cancel(
	c *gin.Context,
) {
	result, err :=
		h.service.Cancel(
			c.Request.Context(),
			c.Param(
				"return_id",
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

func writeError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidInput,
	),
		errors.Is(
			err,
			ErrRejectionReasonRequired,
		),
		errors.Is(
			err,
			ErrInvalidReceivedQuantity,
		),
		errors.Is(
			err,
			ErrInvalidInspection,
		),
		errors.Is(
			err,
			ErrDuplicateReturnItem,
		):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": gin.H{
					"code": "INVALID_RETURN",

					"message": err.Error(),
				},
			},
		)

	case errors.Is(
		err,
		ErrReturnNotFound,
	),
		errors.Is(
			err,
			ErrOrderNotFound,
		),
		errors.Is(
			err,
			ErrReturnItemNotFound,
		):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": gin.H{
					"code": "RETURN_NOT_FOUND",

					"message": err.Error(),
				},
			},
		)

	case errors.Is(
		err,
		ErrReturnNotAllowed,
	),
		errors.Is(
			err,
			ErrReturnQuantityExceeded,
		),
		errors.Is(
			err,
			ErrInvalidReturnTransition,
		):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": gin.H{
					"code": "RETURN_CONFLICT",

					"message": err.Error(),
				},
			},
		)

	default:
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code": "INTERNAL_ERROR",

					"message": "Unable to process return",
				},
			},
		)
	}
}
