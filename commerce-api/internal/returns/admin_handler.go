package returns

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

func (h *AdminHandler) Approve(
	c *gin.Context,
) {
	result, err :=
		h.service.Approve(
			c.Request.Context(),
			c.Param(
				"return_id",
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

func (h *AdminHandler) Reject(
	c *gin.Context,
) {
	var request RejectRequest

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
		h.service.Reject(
			c.Request.Context(),
			c.Param(
				"return_id",
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

func (h *AdminHandler) Receive(
	c *gin.Context,
) {
	var request ReceiveRequest

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
		h.service.Receive(
			c.Request.Context(),
			c.Param(
				"return_id",
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

func (h *AdminHandler) Inspect(
	c *gin.Context,
) {
	var request InspectRequest

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
		h.service.Inspect(
			c.Request.Context(),
			c.Param(
				"return_id",
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
