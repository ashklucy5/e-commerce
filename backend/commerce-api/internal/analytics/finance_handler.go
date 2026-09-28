package analytics

import (
	"net/http"

	"github.com/gin-gonic/gin"

	platformerrors "project.local/commerce-api/internal/platform/errors"
)

func (h *Handler) ProfitLoss(
	c *gin.Context,
) {
	query, ok :=
		h.query(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.ProfitLoss(
			c.Request.Context(),
			query,
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,

			"meta": query.Meta(),
		},
	)
}

func (h *Handler) ProfitLossTrend(
	c *gin.Context,
) {
	query, ok :=
		h.query(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.ProfitLossTrend(
			c.Request.Context(),
			query,
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,

			"meta": query.Meta(),
		},
	)
}

func (h *Handler) ProductProfitability(
	c *gin.Context,
) {
	query, ok :=
		h.query(
			c,
		)
	if !ok {
		return
	}

	limit, ok :=
		analyticsLimit(
			c,
		)
	if !ok {
		return
	}

	result, err :=
		h.service.ProductProfitability(
			c.Request.Context(),
			query,
			limit,
		)
	if err != nil {
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,

			"meta": query.Meta(),
		},
	)
}
