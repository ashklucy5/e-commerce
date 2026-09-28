package refund

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	service *Service
	actorID string
}

func NewAdminHandler(
	service *Service,
	actorID string,
) *AdminHandler {
	return &AdminHandler{
		service: service,
		actorID: strings.TrimSpace(
			actorID,
		),
	}
}

type requestRefundBody struct {
	Reason string `json:"reason"`
}

func (h *AdminHandler) RequestForReturn(
	c *gin.Context,
) {
	var request requestRefundBody

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
		h.service.RequestForReturn(
			c.Request.Context(),
			c.Param(
				"return_id",
			),
			request.Reason,
			h.actorID,
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

func (h *AdminHandler) Approve(
	c *gin.Context,
) {
	result, err :=
		h.service.Approve(
			c.Request.Context(),
			c.Param(
				"refund_id",
			),
			h.actorID,
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

func (h *AdminHandler) StartProcessing(
	c *gin.Context,
) {
	result, err :=
		h.service.StartProcessing(
			c.Request.Context(),
			c.Param(
				"refund_id",
			),
			h.actorID,
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

func (h *AdminHandler) Succeed(
	c *gin.Context,
) {
	var request SuccessRequest

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
		h.service.Succeed(
			c.Request.Context(),
			c.Param(
				"refund_id",
			),
			request,
			h.actorID,
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

func (h *AdminHandler) Fail(
	c *gin.Context,
) {
	var request FailureRequest

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
		h.service.Fail(
			c.Request.Context(),
			c.Param(
				"refund_id",
			),
			request,
			h.actorID,
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
