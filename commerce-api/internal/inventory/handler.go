package inventory

import (
	"errors"
	"net/http"
	"strconv"

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

func (h *Handler) List(
	c *gin.Context,
) {
	limit := parseIntegerQuery(
		c,
		"limit",
		50,
	)

	offset := parseIntegerQuery(
		c,
		"offset",
		0,
	)

	result, err := h.service.List(
		c.Request.Context(),
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
			"data": result,
		},
	)
}

func (h *Handler) Get(
	c *gin.Context,
) {
	result, err := h.service.Get(
		c.Request.Context(),
		c.Param("variant_id"),
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

func (h *Handler) Adjust(
	c *gin.Context,
) {
	var request AdjustRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "invalid request body",
			},
		)

		return
	}

	result, err := h.service.Adjust(
		c.Request.Context(),
		c.Param("variant_id"),
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
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Update(
	c *gin.Context,
) {
	var request UpdateRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "invalid request body",
			},
		)

		return
	}

	result, err := h.service.Update(
		c.Request.Context(),
		c.Param("variant_id"),
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
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Movements(
	c *gin.Context,
) {
	limit := parseIntegerQuery(
		c,
		"limit",
		50,
	)

	offset := parseIntegerQuery(
		c,
		"offset",
		0,
	)

	result, err := h.service.ListMovements(
		c.Request.Context(),
		c.Param("variant_id"),
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
			"data": result,
		},
	)
}

func parseIntegerQuery(
	c *gin.Context,
	name string,
	fallback int,
) int {
	value :=
		c.Query(name)

	if value == "" {
		return fallback
	}

	parsed, err :=
		strconv.Atoi(value)

	if err != nil {
		return fallback
	}

	return parsed
}

func writeError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ErrInvalidInput,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": err.Error(),
			},
		)

	case errors.Is(
		err,
		ErrNotFound,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": err.Error(),
			},
		)

	case errors.Is(
		err,
		ErrInsufficientStock,
	),
		errors.Is(
			err,
			ErrReservationClosed,
		),
		errors.Is(
			err,
			ErrInventoryInvariant,
		):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": err.Error(),
			},
		)

	default:
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "internal server error",
			},
		)
	}
}
