package notification

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/adminauth"
	"project.local/commerce-api/internal/auth"
)

type InboxHandler struct {
	service *Service
}

func NewInboxHandler(
	service *Service,
) *InboxHandler {
	return &InboxHandler{
		service: service,
	}
}

func RegisterCustomerInboxRoutes(
	group *gin.RouterGroup,
	handler *InboxHandler,
	requireAuth gin.HandlerFunc,
) {
	routes :=
		group.Group(
			"/customers/me/notifications",
		)

	routes.Use(
		requireAuth,
	)

	routes.GET(
		"",
		handler.ListCustomer,
	)

	routes.GET(
		"/summary",
		handler.CustomerSummary,
	)

	routes.POST(
		"/read-all",
		handler.MarkAllCustomerRead,
	)

	routes.POST(
		"/:notification_id/read",
		handler.MarkCustomerRead,
	)
}

func RegisterAdminInboxRoutes(
	group *gin.RouterGroup,
	handler *InboxHandler,
) {
	routes :=
		group.Group(
			"/notifications",
		)

	routes.GET(
		"",
		handler.ListStaff,
	)

	routes.GET(
		"/summary",
		handler.StaffSummary,
	)

	routes.POST(
		"/read-all",
		handler.MarkAllStaffRead,
	)

	routes.POST(
		"/:notification_id/read",
		handler.MarkStaffRead,
	)
}

func (h *InboxHandler) ListCustomer(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)

	if !ok {
		writeInboxError(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication required",
		)

		return
	}

	result, err :=
		h.service.ListCustomerInbox(
			c.Request.Context(),
			customerID,
			inboxOptionsFromQuery(c),
		)

	if err != nil {
		writeInboxServiceError(
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
				"limit":  result.Limit,
				"offset": result.Offset,
			},
		},
	)
}

func (h *InboxHandler) CustomerSummary(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)

	if !ok {
		writeInboxError(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication required",
		)

		return
	}

	result, err :=
		h.service.CustomerInboxSummary(
			c.Request.Context(),
			customerID,
		)

	if err != nil {
		writeInboxServiceError(
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

func (h *InboxHandler) MarkCustomerRead(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)

	if !ok {
		writeInboxError(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication required",
		)

		return
	}

	updated, err :=
		h.service.MarkCustomerInboxRead(
			c.Request.Context(),
			customerID,
			c.Param("notification_id"),
		)

	if err != nil {
		writeInboxServiceError(
			c,
			err,
		)

		return
	}

	if !updated {
		writeInboxError(
			c,
			http.StatusNotFound,
			"NOTIFICATION_NOT_FOUND",
			"Notification not found",
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": gin.H{
				"read": true,
			},
		},
	)
}

func (h *InboxHandler) MarkAllCustomerRead(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)

	if !ok {
		writeInboxError(
			c,
			http.StatusUnauthorized,
			"UNAUTHORIZED",
			"Authentication required",
		)

		return
	}

	count, err :=
		h.service.MarkAllCustomerInboxRead(
			c.Request.Context(),
			customerID,
		)

	if err != nil {
		writeInboxServiceError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": gin.H{
				"updated": count,
			},
		},
	)
}

func (h *InboxHandler) ListStaff(
	c *gin.Context,
) {
	staffID, ok :=
		staffIDFromContext(
			c,
		)

	if !ok {
		writeInboxError(
			c,
			http.StatusUnauthorized,
			"ADMIN_AUTHENTICATION_REQUIRED",
			"Admin authentication required",
		)

		return
	}

	result, err :=
		h.service.ListStaffInbox(
			c.Request.Context(),
			staffID,
			inboxOptionsFromQuery(c),
		)

	if err != nil {
		writeInboxServiceError(
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
				"limit":  result.Limit,
				"offset": result.Offset,
			},
		},
	)
}

func (h *InboxHandler) StaffSummary(
	c *gin.Context,
) {
	staffID, ok :=
		staffIDFromContext(
			c,
		)

	if !ok {
		writeInboxError(
			c,
			http.StatusUnauthorized,
			"ADMIN_AUTHENTICATION_REQUIRED",
			"Admin authentication required",
		)

		return
	}

	result, err :=
		h.service.StaffInboxSummary(
			c.Request.Context(),
			staffID,
		)

	if err != nil {
		writeInboxServiceError(
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

func (h *InboxHandler) MarkStaffRead(
	c *gin.Context,
) {
	staffID, ok :=
		staffIDFromContext(
			c,
		)

	if !ok {
		writeInboxError(
			c,
			http.StatusUnauthorized,
			"ADMIN_AUTHENTICATION_REQUIRED",
			"Admin authentication required",
		)

		return
	}

	updated, err :=
		h.service.MarkStaffInboxRead(
			c.Request.Context(),
			staffID,
			c.Param("notification_id"),
		)

	if err != nil {
		writeInboxServiceError(
			c,
			err,
		)

		return
	}

	if !updated {
		writeInboxError(
			c,
			http.StatusNotFound,
			"NOTIFICATION_NOT_FOUND",
			"Notification not found",
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": gin.H{
				"read": true,
			},
		},
	)
}

func (h *InboxHandler) MarkAllStaffRead(
	c *gin.Context,
) {
	staffID, ok :=
		staffIDFromContext(
			c,
		)

	if !ok {
		writeInboxError(
			c,
			http.StatusUnauthorized,
			"ADMIN_AUTHENTICATION_REQUIRED",
			"Admin authentication required",
		)

		return
	}

	count, err :=
		h.service.MarkAllStaffInboxRead(
			c.Request.Context(),
			staffID,
		)

	if err != nil {
		writeInboxServiceError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": gin.H{
				"updated": count,
			},
		},
	)
}

func staffIDFromContext(
	c *gin.Context,
) (
	string,
	bool,
) {
	principal, ok :=
		adminauth.PrincipalFromContext(
			c,
		)

	if !ok ||
		strings.TrimSpace(
			principal.Staff.ID,
		) == "" {

		return "",
			false
	}

	return principal.Staff.ID,
		true
}

func inboxOptionsFromQuery(
	c *gin.Context,
) InboxListOptions {
	limit :=
		parseInboxInteger(
			c.Query("limit"),
			30,
		)

	offset :=
		parseInboxInteger(
			c.Query("offset"),
			0,
		)

	return InboxListOptions{
		Limit:  limit,
		Offset: offset,

		UnreadOnly: strings.EqualFold(
			strings.TrimSpace(
				c.Query("unread"),
			),
			"true",
		) ||
			c.Query("unread") == "1",

		Category: c.Query(
			"category",
		),
	}
}

func parseInboxInteger(
	value string,
	fallback int,
) int {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return fallback
	}

	parsed, err :=
		strconv.Atoi(
			value,
		)

	if err != nil {
		return fallback
	}

	return parsed
}

func writeInboxServiceError(
	c *gin.Context,
	err error,
) {
	if errors.Is(
		err,
		ErrInvalidInput,
	) {
		writeInboxError(
			c,
			http.StatusBadRequest,
			"INVALID_NOTIFICATION_REQUEST",
			"Invalid notification request",
		)

		return
	}

	writeInboxError(
		c,
		http.StatusInternalServerError,
		"NOTIFICATION_ERROR",
		"Unable to process notifications",
	)
}

func writeInboxError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(
		status,
		gin.H{
			"error": gin.H{
				"code":    code,
				"message": message,
			},
		},
	)
}
