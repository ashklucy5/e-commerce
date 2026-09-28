package support

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/staff"
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

func (h *Handler) Me(
	c *gin.Context,
) {
	account, ok :=
		staff.AccountFromContext(
			c,
		)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	result, err :=
		h.service.Me(
			c.Request.Context(),
			account,
		)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Queues(
	c *gin.Context,
) {
	account, ok :=
		staff.AccountFromContext(c)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	queues, err :=
		h.service.Queues(
			c.Request.Context(),
			account.ID,
		)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": queues,
		},
	)
}

func (h *Handler) UpdatePresence(
	c *gin.Context,
) {
	account, ok :=
		staff.AccountFromContext(c)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	var request UpdatePresenceRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeError(
			c,
			ErrInvalidPresence,
		)
		return
	}

	actor, err :=
		h.service.UpdatePresence(
			c.Request.Context(),
			account.ID,
			request,
		)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": actor,
		},
	)
}

func (h *Handler) ListCases(
	c *gin.Context,
) {
	account, ok :=
		staff.AccountFromContext(c)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	limit, err :=
		supportQueryInt(
			c.Query("limit"),
		)
	if err != nil {
		writeError(
			c,
			ErrCaseNotFound,
		)
		return
	}

	offset, err :=
		supportQueryInt(
			c.Query("offset"),
		)
	if err != nil {
		writeError(
			c,
			ErrCaseNotFound,
		)
		return
	}

	items,
		resolvedLimit,
		resolvedOffset,
		err :=
		h.service.ListCases(
			c.Request.Context(),
			account.ID,
			limit,
			offset,
		)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": items,
			"meta": gin.H{
				"limit": resolvedLimit,

				"offset": resolvedOffset,
			},
		},
	)
}

func (h *Handler) GetCase(
	c *gin.Context,
) {
	account, ok :=
		staff.AccountFromContext(c)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	result, err :=
		h.service.GetCase(
			c.Request.Context(),
			account.ID,
			c.Param("id"),
		)
	if err != nil {
		writeError(c, err)
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
	account, ok :=
		staff.AccountFromContext(c)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	limit, err :=
		supportQueryInt(
			c.Query("limit"),
		)
	if err != nil {
		writeError(
			c,
			ErrCaseNotFound,
		)
		return
	}

	offset, err :=
		supportQueryInt(
			c.Query("offset"),
		)
	if err != nil {
		writeError(
			c,
			ErrCaseNotFound,
		)
		return
	}

	items,
		resolvedLimit,
		resolvedOffset,
		err :=
		h.service.ListMessages(
			c.Request.Context(),
			account.ID,
			c.Param("id"),
			limit,
			offset,
		)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": items,
			"meta": gin.H{
				"limit": resolvedLimit,

				"offset": resolvedOffset,
			},
		},
	)
}

func (h *Handler) ClaimCase(
	c *gin.Context,
) {
	account, ok :=
		staff.AccountFromContext(c)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	result, err :=
		h.service.ClaimCase(
			c.Request.Context(),
			account.ID,
			c.Param("id"),
		)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Reply(
	c *gin.Context,
) {
	account, ok :=
		staff.AccountFromContext(c)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	var request ReplyRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeError(
			c,
			ErrInvalidMessage,
		)
		return
	}

	result, err :=
		h.service.Reply(
			c.Request.Context(),
			account.ID,
			c.Param("id"),
			request,
		)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(
		http.StatusCreated,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Resolve(
	c *gin.Context,
) {
	account, ok :=
		staff.AccountFromContext(c)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	result, err :=
		h.service.Resolve(
			c.Request.Context(),
			account.ID,
			c.Param("id"),
		)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Close(
	c *gin.Context,
) {
	account, ok :=
		staff.AccountFromContext(c)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	result, err :=
		h.service.Close(
			c.Request.Context(),
			account.ID,
			c.Param("id"),
		)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func (h *Handler) Escalate(
	c *gin.Context,
) {
	account, ok :=
		staff.AccountFromContext(c)
	if !ok {
		writeError(
			c,
			staff.ErrInvalidAccessToken,
		)
		return
	}

	var request EscalateRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {
		writeError(
			c,
			ErrInvalidQueue,
		)
		return
	}

	result, err :=
		h.service.Escalate(
			c.Request.Context(),
			account.ID,
			c.Param("id"),
			request,
		)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": result,
		},
	)
}

func supportQueryInt(
	value string,
) (int, error) {
	if value == "" {
		return 0, nil
	}

	return strconv.Atoi(value)
}

func writeError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		staff.ErrInvalidAccessToken,
	):
		writeJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_STAFF_ACCESS_TOKEN",
			"Invalid or expired staff access token",
		)

	case errors.Is(
		err,
		ErrNotSupportActor,
	):
		writeJSONError(
			c,
			http.StatusForbidden,
			"NOT_SUPPORT_ACTOR",
			"Staff account is not a support agent",
		)

	case errors.Is(
		err,
		ErrSupportActorDisabled,
	):
		writeJSONError(
			c,
			http.StatusForbidden,
			"SUPPORT_ACTOR_DISABLED",
			"Support agent is disabled",
		)

	case errors.Is(
		err,
		ErrInvalidPresence,
	):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SUPPORT_PRESENCE",
			"Presence must be offline, available, busy, or away",
		)

	case errors.Is(
		err,
		ErrCaseNotFound,
	):
		writeJSONError(
			c,
			http.StatusNotFound,
			"SUPPORT_CASE_NOT_FOUND",
			"Support case was not found",
		)

	case errors.Is(
		err,
		ErrCaseAlreadyClaimed,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"SUPPORT_CASE_ALREADY_CLAIMED",
			"Support case is already claimed by another agent",
		)

	case errors.Is(
		err,
		ErrCaseNotOwned,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"SUPPORT_CASE_NOT_OWNED",
			"Claim the support case before performing this action",
		)

	case errors.Is(
		err,
		ErrCaseNotResolved,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"SUPPORT_CASE_NOT_RESOLVED",
			"Resolve the support case before closing it",
		)

	case errors.Is(
		err,
		ErrCaseClosed,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"SUPPORT_CASE_CLOSED",
			"Closed support cases cannot be modified",
		)

	case errors.Is(
		err,
		ErrActorCapacity,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"SUPPORT_CAPACITY_REACHED",
			"Support agent has reached the active case limit",
		)

	case errors.Is(
		err,
		ErrInvalidMessage,
	):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SUPPORT_MESSAGE",
			"Support message is invalid",
		)

	case errors.Is(
		err,
		ErrInvalidVisibility,
	):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_MESSAGE_VISIBILITY",
			"Visibility must be customer or internal",
		)

	case errors.Is(
		err,
		ErrInvalidQueue,
	):
		writeJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SUPPORT_QUEUE",
			"Support queue is invalid",
		)

	default:
		writeJSONError(
			c,
			http.StatusInternalServerError,
			"SUPPORT_FAILED",
			"Unable to process support request",
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
