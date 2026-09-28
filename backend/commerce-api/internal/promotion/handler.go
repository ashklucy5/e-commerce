package promotion

import (
	"errors"
	"net/http"
	"time"

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

func (h *Handler) Active(
	c *gin.Context,
) {
	result, err :=
		h.service.ListStorefrontActive(
			c.Request.Context(),
			c.Query(
				"currency",
			),
			time.Time{},
		)
	if err != nil {
		if errors.Is(
			err,
			ErrInvalidCurrency,
		) {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": gin.H{
						"code": "INVALID_CURRENCY",

						"message": "Currency must be a 3-letter code",
					},
				},
			)

			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code": "INTERNAL_ERROR",

					"message": "Unable to load promotions",
				},
			},
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

func (h *Handler) ActiveFlashSales(
	c *gin.Context,
) {
	result, err :=
		h.service.ListActiveFlashSales(
			c.Request.Context(),
			c.Query(
				"currency",
			),
			time.Time{},
		)
	if err != nil {
		if errors.Is(
			err,
			ErrInvalidCurrency,
		) {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": gin.H{
						"code":    "INVALID_CURRENCY",
						"message": "Currency must be a 3-letter code",
					},
				},
			)

			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Unable to load flash sales",
				},
			},
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
