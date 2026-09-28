package crm

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"project.local/commerce-api/internal/auth"
	"project.local/commerce-api/internal/platform/storage"
	"project.local/commerce-api/internal/supportattachment"
)

type AttachmentUploadTargetRequest struct {
	CaseID string `json:"case_id,omitempty"`

	Name string `json:"name"`

	MimeType string `json:"mime_type"`

	Size int64 `json:"size"`
}

type AttachmentHandler struct {
	repository  *Repository
	attachments *supportattachment.Service
}

func NewAttachmentHandler(
	repository *Repository,
	attachments *supportattachment.Service,
) *AttachmentHandler {
	return &AttachmentHandler{
		repository:  repository,
		attachments: attachments,
	}
}

func RegisterAttachmentRoutes(
	group *gin.RouterGroup,
	handler *AttachmentHandler,
	requireAuth gin.HandlerFunc,
) {
	attachments :=
		group.Group(
			"/attachments",
		)

	attachments.Use(
		requireAuth,
	)

	attachments.POST(
		"/upload-target",
		handler.CreateUploadTarget,
	)

	attachments.POST(
		"/:id/complete",
		handler.CompleteUpload,
	)

	attachments.GET(
		"/:id",
		handler.GetAttachment,
	)
}

func (h *AttachmentHandler) CreateUploadTarget(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	var request AttachmentUploadTargetRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAttachmentError(
			c,
			ErrInvalidRequest,
		)

		return
	}

	caseID :=
		strings.TrimSpace(
			request.CaseID,
		)

	/*
		CaseID is optional because a customer may upload
		an image before creating a brand-new support case.

		When supplied, prove that the authenticated customer
		owns the case before creating a case-scoped object.
	*/
	if caseID != "" {
		if !validUUID(
			caseID,
		) {
			writeAttachmentError(
				c,
				ErrInvalidRequest,
			)

			return
		}

		current, err :=
			h.repository.GetCustomerCase(
				c.Request.Context(),
				customerID,
				caseID,
			)
		if err != nil {
			writeAttachmentError(
				c,
				err,
			)

			return
		}

		if current.Status ==
			StatusClosed {

			writeAttachmentError(
				c,
				ErrCaseClosed,
			)

			return
		}
	}

	result, err :=
		h.attachments.CreateUploadTarget(
			c.Request.Context(),
			supportattachment.CreateUploadRequest{
				CaseID: caseID,

				UploaderType: supportattachment.
					UploaderCustomer,

				OwnerID: customerID,

				OriginalFilename: request.Name,

				MimeType: request.MimeType,

				ByteSize: request.Size,
			},
		)
	if err != nil {
		writeAttachmentError(
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
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeError(
			c,
			auth.ErrInvalidAccessToken,
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
				UploaderCustomer,
			customerID,
		)
	if err != nil {
		writeAttachmentError(
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

func (h *AttachmentHandler) GetAttachment(
	c *gin.Context,
) {
	customerID, ok :=
		auth.CustomerIDFromContext(
			c,
		)
	if !ok {
		writeError(
			c,
			auth.ErrInvalidAccessToken,
		)

		return
	}

	attachmentID :=
		strings.TrimSpace(
			c.Param("id"),
		)

	/*
		Do not expose whether malformed/random attachment IDs
		correspond to private objects.
	*/
	if !validUUID(
		attachmentID,
	) {
		writeCustomerAttachmentNotFound(
			c,
		)

		return
	}

	allowed, err :=
		h.repository.CustomerCanAccessAttachment(
			c.Request.Context(),
			customerID,
			attachmentID,
		)
	if err != nil {
		writeAttachmentError(
			c,
			err,
		)

		return
	}

	if !allowed {
		writeCustomerAttachmentNotFound(
			c,
		)

		return
	}

	attachment, err :=
		h.attachments.GetAttachment(
			c.Request.Context(),
			attachmentID,
		)
	if err != nil {
		if errors.Is(
			err,
			supportattachment.ErrNotFound,
		) ||
			errors.Is(
				err,
				supportattachment.
					ErrInvalidAttachmentID,
			) {

			writeCustomerAttachmentNotFound(
				c,
			)

			return
		}

		writeAttachmentError(
			c,
			err,
		)

		return
	}

	/*
		The metadata row intentionally remains after B2 deletion.

		This lets an authorized customer receive an explicit
		expired response instead of a broken image.
	*/
	if attachment.Status ==
		supportattachment.StatusDeleted ||
		attachment.DeletedAt != nil {

		writeJSONError(
			c,
			http.StatusGone,
			"SUPPORT_ATTACHMENT_EXPIRED",
			"This support attachment is no longer available",
		)

		return
	}

	if attachment.Status !=
		supportattachment.StatusAttached {

		writeCustomerAttachmentNotFound(
			c,
		)

		return
	}

	target, err :=
		h.attachments.CreateDownload(
			c.Request.Context(),
			attachment,
		)
	if err != nil {
		writeAttachmentError(
			c,
			err,
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": target,
		},
	)
}

func writeCustomerAttachmentNotFound(
	c *gin.Context,
) {
	writeJSONError(
		c,
		http.StatusNotFound,
		"SUPPORT_ATTACHMENT_NOT_FOUND",
		"Support attachment was not found",
	)
}

func writeAttachmentError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		auth.ErrInvalidAccessToken,
	):
		writeJSONError(
			c,
			http.StatusUnauthorized,
			"INVALID_ACCESS_TOKEN",
			"Invalid or expired access token",
		)

	case errors.Is(
		err,
		ErrInvalidRequest,
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
			err.Error(),
		)

	case errors.Is(
		err,
		ErrCaseNotFound,
	):
		writeJSONError(
			c,
			http.StatusNotFound,
			"CRM_CASE_NOT_FOUND",
			"Customer service case was not found",
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
		ErrCaseClosed,
	):
		writeJSONError(
			c,
			http.StatusConflict,
			"CRM_CASE_CLOSED",
			"Closed customer service cases cannot receive new attachments",
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
