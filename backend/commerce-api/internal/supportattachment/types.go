package supportattachment

import (
	"errors"
	"time"

	"project.local/commerce-api/internal/platform/storage"
)

const (
	MaxImageBytes int64 = 5 * 1024 * 1024

	DownloadURLTTL = 5 * time.Minute
)

type UploaderType string

const (
	UploaderCustomer UploaderType = "customer"
	UploaderSupport  UploaderType = "support"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusReady    Status = "ready"
	StatusAttached Status = "attached"
	StatusDeleted  Status = "deleted"
)

var (
	ErrNotFound = errors.New(
		"support attachment not found",
	)

	ErrInvalidState = errors.New(
		"support attachment is not in the required state",
	)

	ErrInvalidAttachmentID = errors.New(
		"invalid support attachment id",
	)

	ErrInvalidUploader = errors.New(
		"invalid support attachment uploader",
	)

	ErrInvalidOwner = errors.New(
		"invalid support attachment owner",
	)

	ErrInvalidCaseID = errors.New(
		"invalid support case id",
	)

	ErrInvalidFilename = errors.New(
		"invalid support attachment filename",
	)

	ErrInvalidContentType = errors.New(
		"support attachment must be JPEG, PNG, WebP, or GIF",
	)

	ErrInvalidImageSize = errors.New(
		"support attachment must be between 1 byte and 5 MB",
	)

	ErrUploadNotFound = errors.New(
		"uploaded support attachment was not found",
	)

	ErrUploadedObjectMismatch = errors.New(
		"uploaded support attachment does not match the expected file",
	)
)

type Attachment struct {
	ID string `json:"id"`

	CaseID    string `json:"case_id,omitempty"`
	MessageID string `json:"message_id,omitempty"`

	UploaderType UploaderType `json:"uploader_type"`

	CustomerID     string `json:"customer_id,omitempty"`
	SupportActorID string `json:"support_actor_id,omitempty"`

	StorageKey string `json:"-"`

	OriginalFilename string `json:"original_filename"`
	MimeType         string `json:"mime_type"`
	ByteSize         int64  `json:"byte_size"`

	Status Status `json:"status"`

	UploadedAt *time.Time `json:"uploaded_at,omitempty"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreatePendingInput struct {
	ID string

	CaseID string

	UploaderType UploaderType

	CustomerID     string
	SupportActorID string

	StorageKey string

	OriginalFilename string
	MimeType         string
	ByteSize         int64
}

type CreateUploadRequest struct {
	CaseID string

	UploaderType UploaderType

	OwnerID string

	OriginalFilename string

	MimeType string

	ByteSize int64
}

type UploadSession struct {
	Attachment Attachment `json:"attachment"`

	Upload storage.UploadTarget `json:"upload"`
}

type DownloadTarget struct {
	ID string `json:"id"`

	OriginalFilename string `json:"original_filename"`

	MimeType string `json:"mime_type"`

	ByteSize int64 `json:"byte_size"`

	URL string `json:"url"`

	ExpiresAt time.Time `json:"expires_at"`
}

type RetentionCandidate struct {
	ID         string
	StorageKey string
}
