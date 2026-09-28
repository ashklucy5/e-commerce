package search

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Suggestions(
	c *gin.Context,
) {
	limit, err := optionalQueryInt(
		c.Query("limit"),
	)
	if err != nil {
		writeSearchError(
			c,
			ErrInvalidLimit,
		)
		return
	}

	result, err := h.service.Suggestions(
		c.Request.Context(),
		c.Query("q"),
		limit,
	)
	if err != nil {
		writeSearchError(
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
