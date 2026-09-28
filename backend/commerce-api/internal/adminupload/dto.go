package adminupload

import (
	"project.local/commerce-api/internal/platform/storage"
)

type CreateUploadRequest struct {
	Filename      string          `json:"filename"`
	ContentType   string          `json:"content_type"`
	ContentLength int64           `json:"content_length"`
	Purpose       storage.Purpose `json:"purpose"`
}

type CreateUploadResponse struct {
	Filename string               `json:"filename"`
	Purpose  storage.Purpose      `json:"purpose"`
	Upload   storage.UploadTarget `json:"upload"`
}
