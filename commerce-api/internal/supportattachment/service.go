package supportattachment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"project.local/commerce-api/internal/platform/storage"
)

type serviceRepository interface {
	CreatePending(
		ctx context.Context,
		input CreatePendingInput,
	) (Attachment, error)

	GetByID(
		ctx context.Context,
		attachmentID string,
	) (Attachment, error)

	MarkReady(
		ctx context.Context,
		attachmentID string,
		uploaderType UploaderType,
		ownerID string,
	) (Attachment, error)
}

type Service struct {
	repository serviceRepository
	storage    *storage.Gateway
}

func NewService(
	repository serviceRepository,
	storageGateway *storage.Gateway,
) *Service {
	return &Service{
		repository: repository,
		storage:    storageGateway,
	}
}

func (s *Service) CreateUploadTarget(
	ctx context.Context,
	request CreateUploadRequest,
) (UploadSession, error) {
	uploaderType, err :=
		validateUploaderType(
			request.UploaderType,
		)
	if err != nil {
		return UploadSession{}, err
	}

	ownerID, ok :=
		normalizeUUID(
			request.OwnerID,
		)
	if !ok {
		return UploadSession{},
			ErrInvalidOwner
	}

	caseID := ""

	if strings.TrimSpace(
		request.CaseID,
	) != "" {
		var valid bool

		caseID, valid =
			normalizeUUID(
				request.CaseID,
			)

		if !valid {
			return UploadSession{},
				ErrInvalidCaseID
		}
	}

	filename, err :=
		normalizeFilename(
			request.OriginalFilename,
		)
	if err != nil {
		return UploadSession{}, err
	}

	contentType :=
		normalizeContentType(
			request.MimeType,
		)

	extension, ok :=
		imageExtension(
			contentType,
		)
	if !ok {
		return UploadSession{},
			ErrInvalidContentType
	}

	if request.ByteSize <= 0 ||
		request.ByteSize > MaxImageBytes {
		return UploadSession{},
			ErrInvalidImageSize
	}

	attachmentID, err :=
		newUUID()
	if err != nil {
		return UploadSession{},
			fmt.Errorf(
				"generate support attachment id: %w",
				err,
			)
	}

	storageKey :=
		supportStorageKey(
			caseID,
			attachmentID,
			extension,
		)

	target, err :=
		s.storage.CreateUpload(
			ctx,
			storage.UploadRequest{
				Key: storageKey,

				ContentType: contentType,

				ContentLength: request.ByteSize,

				Access: storage.AccessPrivate,

				Purpose: storage.PurposeSupportImage,

				CacheControl: "private, no-store",
			},
		)
	if err != nil {
		return UploadSession{}, err
	}

	createInput :=
		CreatePendingInput{
			ID: attachmentID,

			CaseID: caseID,

			UploaderType: uploaderType,

			StorageKey: storageKey,

			OriginalFilename: filename,

			MimeType: contentType,

			ByteSize: request.ByteSize,
		}

	switch uploaderType {
	case UploaderCustomer:
		createInput.CustomerID =
			ownerID

	case UploaderSupport:
		createInput.SupportActorID =
			ownerID
	}

	attachment, err :=
		s.repository.CreatePending(
			ctx,
			createInput,
		)
	if err != nil {
		return UploadSession{}, err
	}

	return UploadSession{
		Attachment: attachment,

		Upload: target,
	}, nil
}

func (s *Service) CompleteUpload(
	ctx context.Context,
	attachmentID string,
	uploaderType UploaderType,
	ownerID string,
) (Attachment, error) {
	attachmentID, ok :=
		normalizeUUID(
			attachmentID,
		)
	if !ok {
		return Attachment{},
			ErrInvalidAttachmentID
	}

	uploaderType, err :=
		validateUploaderType(
			uploaderType,
		)
	if err != nil {
		return Attachment{}, err
	}

	ownerID, ok =
		normalizeUUID(
			ownerID,
		)
	if !ok {
		return Attachment{},
			ErrInvalidOwner
	}

	attachment, err :=
		s.repository.GetByID(
			ctx,
			attachmentID,
		)
	if err != nil {
		return Attachment{}, err
	}

	if !attachmentOwnedBy(
		attachment,
		uploaderType,
		ownerID,
	) {
		return Attachment{},
			ErrNotFound
	}

	switch attachment.Status {
	case StatusReady:
		return attachment, nil

	case StatusPending:
		// Continue below.

	default:
		return Attachment{},
			ErrInvalidState
	}

	objectInfo, err :=
		s.storage.Stat(
			ctx,
			attachment.StorageKey,
		)
	if err != nil {
		if errors.Is(
			err,
			storage.ErrNotFound,
		) {
			return Attachment{},
				ErrUploadNotFound
		}

		return Attachment{}, err
	}

	if objectInfo.ContentLength <= 0 ||
		objectInfo.ContentLength >
			MaxImageBytes {
		return Attachment{},
			ErrInvalidImageSize
	}

	if objectInfo.ContentLength !=
		attachment.ByteSize {
		return Attachment{},
			ErrUploadedObjectMismatch
	}

	actualContentType :=
		normalizeContentType(
			objectInfo.ContentType,
		)

	if actualContentType == "" ||
		actualContentType !=
			attachment.MimeType {
		return Attachment{},
			ErrUploadedObjectMismatch
	}

	expectedExtension, ok :=
		imageExtension(
			actualContentType,
		)
	if !ok {
		return Attachment{},
			ErrInvalidContentType
	}

	if !strings.EqualFold(
		path.Ext(
			attachment.StorageKey,
		),
		expectedExtension,
	) {
		return Attachment{},
			ErrUploadedObjectMismatch
	}

	return s.repository.MarkReady(
		ctx,
		attachmentID,
		uploaderType,
		ownerID,
	)
}

func (s *Service) GetAttachment(
	ctx context.Context,
	attachmentID string,
) (Attachment, error) {
	attachmentID, ok :=
		normalizeUUID(
			attachmentID,
		)
	if !ok {
		return Attachment{},
			ErrInvalidAttachmentID
	}

	return s.repository.GetByID(
		ctx,
		attachmentID,
	)
}

func (s *Service) CreateDownload(
	ctx context.Context,
	attachment Attachment,
) (DownloadTarget, error) {
	if attachment.Status !=
		StatusAttached {
		return DownloadTarget{},
			ErrInvalidState
	}

	if attachment.DeletedAt != nil {
		return DownloadTarget{},
			ErrInvalidState
	}

	if !strings.HasPrefix(
		attachment.StorageKey,
		"private/support/",
	) {
		return DownloadTarget{},
			ErrInvalidState
	}

	target, err :=
		s.storage.CreateDownload(
			ctx,
			storage.DownloadRequest{
				Key: attachment.StorageKey,

				ExpiresIn: DownloadURLTTL,
			},
		)
	if err != nil {
		return DownloadTarget{}, err
	}

	return DownloadTarget{
		ID: attachment.ID,

		OriginalFilename: attachment.OriginalFilename,

		MimeType: attachment.MimeType,

		ByteSize: attachment.ByteSize,

		URL: target.URL,

		ExpiresAt: target.ExpiresAt,
	}, nil
}

func attachmentOwnedBy(
	attachment Attachment,
	uploaderType UploaderType,
	ownerID string,
) bool {
	if attachment.UploaderType !=
		uploaderType {
		return false
	}

	switch uploaderType {
	case UploaderCustomer:
		return attachment.CustomerID ==
			ownerID

	case UploaderSupport:
		return attachment.SupportActorID ==
			ownerID

	default:
		return false
	}
}

func validateUploaderType(
	value UploaderType,
) (UploaderType, error) {
	switch value {
	case UploaderCustomer,
		UploaderSupport:
		return value, nil

	default:
		return "",
			ErrInvalidUploader
	}
}

func normalizeContentType(
	value string,
) string {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if separator :=
		strings.Index(
			value,
			";",
		); separator >= 0 {
		value =
			strings.TrimSpace(
				value[:separator],
			)
	}

	return value
}

func imageExtension(
	contentType string,
) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true

	case "image/png":
		return ".png", true

	case "image/webp":
		return ".webp", true

	case "image/gif":
		return ".gif", true

	default:
		return "", false
	}
}

func normalizeFilename(
	value string,
) (string, error) {
	value =
		strings.TrimSpace(
			value,
		)

	value =
		strings.ReplaceAll(
			value,
			"\\",
			"/",
		)

	value =
		path.Base(
			value,
		)

	value =
		strings.TrimSpace(
			value,
		)

	if value == "" ||
		value == "." ||
		value == ".." {
		return "",
			ErrInvalidFilename
	}

	if strings.ContainsRune(
		value,
		0,
	) {
		return "",
			ErrInvalidFilename
	}

	if utf8.RuneCountInString(
		value,
	) > 255 {
		return "",
			ErrInvalidFilename
	}

	return value, nil
}

func supportStorageKey(
	caseID string,
	attachmentID string,
	extension string,
) string {
	if caseID == "" {
		return fmt.Sprintf(
			"private/support/staging/%s%s",
			attachmentID,
			extension,
		)
	}

	return fmt.Sprintf(
		"private/support/%s/%s%s",
		caseID,
		attachmentID,
		extension,
	)
}

func normalizeUUID(
	value string,
) (string, bool) {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if len(value) != 36 {
		return "", false
	}

	for _, index := range []int{
		8,
		13,
		18,
		23,
	} {

		if value[index] != '-' {
			return "", false
		}
	}

	compact :=
		strings.ReplaceAll(
			value,
			"-",
			"",
		)

	if len(compact) != 32 {
		return "", false
	}

	if _, err :=
		hex.DecodeString(
			compact,
		); err != nil {
		return "", false
	}

	return value, true
}

func newUUID() (
	string,
	error,
) {
	value :=
		make(
			[]byte,
			16,
		)

	if _, err :=
		rand.Read(
			value,
		); err != nil {
		return "", err
	}

	// RFC 4122 version 4.
	value[6] =
		(value[6] & 0x0f) |
			0x40

	// RFC 4122 variant.
	value[8] =
		(value[8] & 0x3f) |
			0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		value[0:4],
		value[4:6],
		value[6:8],
		value[8:10],
		value[10:16],
	), nil
}
