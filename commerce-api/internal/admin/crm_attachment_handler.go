package admin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	platformerrors "project.local/commerce-api/internal/platform/errors"
	platformstorage "project.local/commerce-api/internal/platform/storage"
	"project.local/commerce-api/internal/support"
	"project.local/commerce-api/internal/supportattachment"
)

type CRMAttachmentHandler struct {
	db          *pgxpool.Pool
	attachments *supportattachment.Service
}

type CRMCreateAttachmentUploadRequest struct {
	Name string `json:"name"`

	MimeType string `json:"mime_type"`

	Size int64 `json:"size"`
}

func NewCRMAttachmentHandler(
	db *pgxpool.Pool,
	storageGateway *platformstorage.Gateway,
) *CRMAttachmentHandler {
	repository :=
		supportattachment.NewRepository(
			db,
		)

	return &CRMAttachmentHandler{
		db: db,

		attachments: supportattachment.NewService(
			repository,
			storageGateway,
		),
	}
}

func (h *CRMAttachmentHandler) CreateUploadTarget(
	c *gin.Context,
) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(
			c,
		)
	if !ok {
		return
	}

	caseID :=
		strings.TrimSpace(
			c.Param("id"),
		)

	var request CRMCreateAttachmentUploadRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		writeAdminCRMAttachmentError(
			c,
			support.ErrInvalidMessage,
		)

		return
	}

	adminService :=
		support.NewAdminService(
			h.db,
		)

	actor, err :=
		adminService.
			AttachmentActorForUpload(
				c.Request.Context(),
				principal.Staff.ID,
				queueScoped,
				caseID,
			)
	if err != nil {
		writeAdminCRMAttachmentError(
			c,
			err,
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
		writeAdminCRMAttachmentError(
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

func (h *CRMAttachmentHandler) CompleteUpload(
	c *gin.Context,
) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(
			c,
		)
	if !ok {
		return
	}

	adminService :=
		support.NewAdminService(
			h.db,
		)

	actor, err :=
		adminService.AttachmentActor(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
		)
	if err != nil {
		writeAdminCRMAttachmentError(
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
		writeAdminCRMAttachmentError(
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

func (h *CRMAttachmentHandler) GetAttachment(
	c *gin.Context,
) {
	principal, queueScoped, ok :=
		adminCRMPrincipal(
			c,
		)
	if !ok {
		return
	}

	attachmentID :=
		strings.TrimSpace(
			c.Param("id"),
		)

	adminService :=
		support.NewAdminService(
			h.db,
		)

	allowed, err :=
		adminService.CanAccessAttachment(
			c.Request.Context(),
			principal.Staff.ID,
			queueScoped,
			attachmentID,
		)
	if err != nil {
		writeAdminCRMAttachmentError(
			c,
			err,
		)

		return
	}

	/*
		Do not reveal whether a private attachment exists
		when this Admin principal cannot access its CRM case.
	*/
	if !allowed {
		writeAdminCRMAttachmentNotFound(
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

			writeAdminCRMAttachmentNotFound(
				c,
			)

			return
		}

		writeAdminCRMAttachmentError(
			c,
			err,
		)

		return
	}

	/*
		Deletion keeps the PostgreSQL tombstone so CRM history
		can show an intentional expired-attachment state.
	*/
	if attachment.Status ==
		supportattachment.StatusDeleted ||
		attachment.DeletedAt != nil {

		platformerrors.Write(
			c,
			platformerrors.New(
				http.StatusGone,
				"ADMIN_CRM_ATTACHMENT_EXPIRED",
				"This support attachment is no longer available",
			),
		)

		return
	}

	if attachment.Status !=
		supportattachment.StatusAttached {

		writeAdminCRMAttachmentNotFound(
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
		writeAdminCRMAttachmentError(
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

func writeAdminCRMAttachmentNotFound(
	c *gin.Context,
) {
	platformerrors.Write(
		c,
		platformerrors.NotFound(
			"ADMIN_CRM_ATTACHMENT_NOT_FOUND",
			"Support attachment was not found",
		),
	)
}

func writeAdminCRMAttachmentError(
	c *gin.Context,
	err error,
) {
	switch {
	/*
		Keep all existing CRM actor/case errors consistent
		with the normal Admin CRM endpoints.
	*/
	case errors.Is(
		err,
		support.ErrNotSupportActor,
	),
		errors.Is(
			err,
			support.ErrSupportActorDisabled,
		),
		errors.Is(
			err,
			support.ErrCaseNotFound,
		),
		errors.Is(
			err,
			support.ErrCaseAlreadyClaimed,
		),
		errors.Is(
			err,
			support.ErrCaseNotOwned,
		),
		errors.Is(
			err,
			support.ErrCaseClosed,
		),
		errors.Is(
			err,
			support.ErrActorCapacity,
		):

		writeAdminCRMError(
			c,
			err,
		)

	case errors.Is(
		err,
		support.ErrInvalidMessage,
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

		platformerrors.Write(
			c,
			platformerrors.BadRequest(
				"INVALID_ADMIN_CRM_ATTACHMENT",
				"Support attachment is invalid",
			),
		)

	case errors.Is(
		err,
		supportattachment.
			ErrNotFound,
	):
		writeAdminCRMAttachmentNotFound(
			c,
		)

	case errors.Is(
		err,
		supportattachment.
			ErrUploadNotFound,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"ADMIN_CRM_ATTACHMENT_UPLOAD_INCOMPLETE",
				"Support attachment upload has not completed",
			),
		)

	case errors.Is(
		err,
		supportattachment.
			ErrUploadedObjectMismatch,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"ADMIN_CRM_ATTACHMENT_UPLOAD_MISMATCH",
				"Uploaded support image does not match the requested file",
			),
		)

	case errors.Is(
		err,
		supportattachment.
			ErrInvalidState,
	):
		platformerrors.Write(
			c,
			platformerrors.Conflict(
				"ADMIN_CRM_ATTACHMENT_INVALID_STATE",
				"Support attachment is not available for this operation",
			),
		)

	case errors.Is(
		err,
		platformstorage.ErrDisabled,
	):
		platformerrors.Write(
			c,
			platformerrors.ServiceUnavailable(
				err,
			),
		)

	default:
		platformerrors.Write(
			c,
			platformerrors.Internal(
				err,
			),
		)
	}
}
