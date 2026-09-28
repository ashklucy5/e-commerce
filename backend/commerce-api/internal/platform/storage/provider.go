package storage

import "context"

type Provider interface {
	Name() string

	CreateUpload(
		ctx context.Context,
		request UploadRequest,
	) (UploadTarget, error)

	CreateDownload(
		ctx context.Context,
		request DownloadRequest,
	) (DownloadTarget, error)

	PublicURL(
		key string,
	) (string, error)

	Stat(
		ctx context.Context,
		key string,
	) (ObjectInfo, error)

	Delete(
		ctx context.Context,
		key string,
	) error
}
