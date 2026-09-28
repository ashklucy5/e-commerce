package storage

import "time"

type Access string

const (
	AccessPublic  Access = "public"
	AccessPrivate Access = "private"
)

type Purpose string

const (
	PurposeProductImage Purpose = "product_image"

	PurposeProduct360Frame Purpose = "product_360_frame"

	PurposeProduct3DModel Purpose = "product_3d_model"

	PurposeCustomerAvatar Purpose = "customer_avatar"

	PurposeSupportImage Purpose = "support_image"

	PurposeTryOnInput Purpose = "tryon_input"

	PurposeTryOnOutput Purpose = "tryon_output"

	PurposeImportAsset Purpose = "import_asset"
)

type UploadRequest struct {
	Key           string
	ContentType   string
	ContentLength int64
	Access        Access
	Purpose       Purpose
	CacheControl  string
}

type UploadTarget struct {
	Provider string `json:"provider"`

	Key string `json:"key"`

	Method string `json:"method"`

	URL string `json:"url"`

	Headers map[string]string `json:"headers,omitempty"`

	ExpiresAt time.Time `json:"expires_at"`
}

type DownloadRequest struct {
	Key string

	ExpiresIn time.Duration
}

type DownloadTarget struct {
	Provider string `json:"provider"`

	Key string `json:"key"`

	URL string `json:"url"`

	ExpiresAt time.Time `json:"expires_at"`
}

type ObjectInfo struct {
	Key string

	ContentType string

	ContentLength int64

	ETag string

	LastModified time.Time
}
