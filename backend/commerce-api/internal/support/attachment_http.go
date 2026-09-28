package support

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/platform/storage"
	"project.local/commerce-api/internal/staff"
	"project.local/commerce-api/internal/supportattachment"
)

type AttachmentUploadTargetRequest struct {
	Name string `json:"name"`

	MimeType string `json:"mime_type"`

	Size int64 `json:"size"`
}

type AttachmentHandler struct {
	service     *Service
	attachments *supportattachment.Service
}

func NewAttachmentHandler(
	service *Service,
	attachments *supportattachment.Service,
) *AttachmentHandler {
	return &AttachmentHandler{
		service:     service,
		attachments: attachments,
	}
}

func RegisterAttachmentRoutes(
	group *gin.RouterGroup,
	handler *AttachmentHandler,
	staffService *staff.Service,
) {
	protected :=
		group.Group("")

	protected.Use(
		staff.RequireAuth(
			staffService,
		),
	)

	protected.Use(
		staff.RequirePermission(
			PermissionPanelAccess,
		),
	)

	protected.POST(
		"/cases/:id/attachments/upload-target",
		staff.RequirePermission(
			PermissionCaseReply,
		),
		handler.CreateUploadTarget,
	)

	protected.POST(
		"/attachments/:id/complete",
		staff.RequirePermission(
			PermissionCaseReply,
		),
		handler.CompleteUpload,
	)
}

func (h *AttachmentHandler) CreateUploadTarget(
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

	caseID :=
		strings.TrimSpace(
			c.Param("id"),
		)

	if !validSupportUUID(
		caseID,
	) {
		writeError(
			c,
			ErrCaseNotFound,
		)

		return
	}

	var request AttachmentUploadTargetRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeSupportAttachmentError(
			c,
			ErrInvalidMessage,
		)

		return
	}

	actor, err :=
		h.service.getActiveActor(
			c.Request.Context(),
			account.ID,
		)
	if err != nil {
		writeSupportAttachmentError(
			c,
			err,
		)

		return
	}

	current, err :=
		h.service.repository.GetCaseForActor(
			c.Request.Context(),
			actor.ID,
			caseID,
		)
	if err != nil {
		writeSupportAttachmentError(
			c,
			err,
		)

		return
	}

	if current.Status == "closed" {
		writeSupportAttachmentError(
			c,
			ErrCaseClosed,
		)

		return
	}

	/*
		Uploading a reply image follows the same ownership
		rule as actually sending the reply.

		Queue membership alone allows reading the ticket,
		but the agent must claim/own it before creating
		new support-side attachment objects.
	*/
	if current.Assignment.AssignedActor == nil ||
		current.Assignment.AssignedActor.ID !=
			actor.ID {

		writeSupportAttachmentError(
			c,
			ErrCaseNotOwned,
		)

		return
	}

	result, err :=
		h.attachments.CreateUploadTarget(
			c.Request.Context(),
			supportattachment.CreateUploadRequest{
				CaseID: caseID,

				UploaderType: supportattachment.
					UploaderSupport,

				OwnerID: actor.ID,

				OriginalFilename: request.Name,

				MimeType: request.MimeType,

				ByteSize: request.Size,
			},
		)
	if err != nil {
		writeSupportAttachmentError(
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

func (h *AttachmentHandler) CompleteUpload(
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

	actor, err :=
		h.service.getActiveActor(
			c.Request.Context(),
			account.ID,
		)
	if err != nil {
		writeSupportAttachmentError(
			c,
			err,
		)

		return
	}

	attachmentID :=
		strings.TrimSpace(
			c.Param("id"),
		)

	result, err :=
		h.attachments.CompleteUpload(
			c.Request.Context(),
			attachmentID,
			supportattachment.
				UploaderSupport,
			actor.ID,
		)
	if err != nil {
		writeSupportAttachmentError(
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

func writeSupportAttachmentError(
	c *gin.Context,
	err error,
) {
	/*
		Reuse the support domain's existing error surface
		for authentication, assignment, and case errors.
	*/
	switch {
	case errors.Is(
		err,
		staff.ErrInvalidAccessToken,
	),
		errors.Is(
			err,
			ErrNotSupportActor,
		),
		errors.Is(
			err,
			ErrSupportActorDisabled,
		),
		errors.Is(
			err,
			ErrCaseNotFound,
		),
		errors.Is(
			err,
			ErrCaseNotOwned,
		),
		errors.Is(
			err,
			ErrCaseClosed,
		):
		writeError(
			c,
			err,
		)

	case errors.Is(
		err,
		ErrInvalidMessage,
	),
		errors.Is(
			err,
			supportattachment.
				ErrInvalidAttachmentID,
		),
		errors.Is(
			err,
			supportattachment.
				ErrInvalidUploader,
		),
		errors.Is(
			err,
			supportattachment.
				ErrInvalidOwner,
		),
		errors.Is(
			err,
			supportattachment.
				ErrInvalidCaseID,
		),
		errors.Is(
			err,
			supportattachment.
				ErrInvalidFilename,
		),
		errors.Is(
			err,
			supportattachment.
				ErrInvalidContentType,
		),
		errors.Is(
			err,
			supportattachment.
				ErrInvalidImageSize,
		):

		writeJSONError(
			c,
			http.StatusBadRequest,
			"INVALID_SUPPORT_ATTACHMENT",
			"Support attachment is invalid",
		)

	case errors.Is(
		err,
		supportattachment.
			ErrNotFound,
	):
		writeJSONError(
			c,
			http.StatusNotFound,
			"SUPPORT_ATTACHMENT_NOT_FOUND",
			"Support attachment was not found",
		)

	case errors.Is(
		err,
		supportattachment.
			ErrUploadNotFound,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"SUPPORT_ATTACHMENT_UPLOAD_INCOMPLETE",
			"Support attachment upload has not completed",
		)

	case errors.Is(
		err,
		supportattachment.
			ErrUploadedObjectMismatch,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"SUPPORT_ATTACHMENT_UPLOAD_MISMATCH",
			"Uploaded support image does not match the requested file",
		)

	case errors.Is(
		err,
		supportattachment.
			ErrInvalidState,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"SUPPORT_ATTACHMENT_INVALID_STATE",
			"Support attachment is not available for this operation",
		)

	case errors.Is(
		err,
		storage.ErrDisabled,
	):
		writeJSONError(
			c,
			http.StatusServiceUnavailable,
			"STORAGE_UNAVAILABLE",
			"Support image storage is unavailable",
		)

	default:
		writeJSONError(
			c,
			http.StatusInternalServerError,
			"SUPPORT_ATTACHMENT_FAILED",
			"Unable to process support attachment",
		)
	}
}
