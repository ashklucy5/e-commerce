package recommendation

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	platformerrors "project.local/commerce-api/internal/platform/errors"
)

type Handler struct {
	telemetry *TelemetryService
}

func NewHandler(telemetry *TelemetryService) *Handler {
	return &Handler{telemetry: telemetry}
}

type recordEventRequest struct {
	SessionID    string `json:"session_id"`
	ProductID    string `json:"product_id" binding:"required"`
	Placement    string `json:"placement" binding:"required"`
	EventType    string `json:"event_type" binding:"required"`
	Strategy     string `json:"strategy" binding:"required"`
	RankPosition *int   `json:"rank_position"`
}

func (h *Handler) RecordEvent(c *gin.Context) {
	var request recordEventRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		platformerrors.Write(c, platformerrors.BadRequest(
			"INVALID_RECOMMENDATION_EVENT",
			"Invalid recommendation event payload",
		))
		return
	}

	customerID, _ := auth.CustomerIDFromContext(c)
	event, err := h.telemetry.RecordEvent(c.Request.Context(), EventInput{
		CustomerID:   customerID,
		SessionID:    request.SessionID,
		ProductID:    request.ProductID,
		Placement:    request.Placement,
		EventType:    request.EventType,
		Strategy:     request.Strategy,
		RankPosition: request.RankPosition,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidEventInput):
			platformerrors.Write(c, platformerrors.BadRequest(
				"INVALID_RECOMMENDATION_EVENT",
				err.Error(),
			))
		case errors.Is(err, ErrProductNotFound):
			platformerrors.Write(c, platformerrors.NotFound(
				"RECOMMENDATION_PRODUCT_NOT_FOUND",
				"Recommended product was not found",
			))
		default:
			platformerrors.Write(c, platformerrors.Internal(err))
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": event})
}
