package storage

import (
	"context"
	"fmt"
	"strings"
)

type Gateway struct {
	provider Provider
}

func NewGateway(
	provider Provider,
) *Gateway {
	if provider == nil {
		provider = NewDisabledProvider()
	}

	return &Gateway{
		provider: provider,
	}
}

func (g *Gateway) ProviderName() string {
	return g.provider.Name()
}

func (g *Gateway) CreateUpload(
	ctx context.Context,
	request UploadRequest,
) (UploadTarget, error) {
	request.Key = normalizeObjectKey(
		request.Key,
	)

	if request.Key == "" {
		return UploadTarget{},
			ErrInvalidKey
	}

	if request.ContentLength < 0 {
		return UploadTarget{},
			fmt.Errorf(
				"content length cannot be negative",
			)
	}

	switch request.Access {
	case AccessPublic:
		if !strings.HasPrefix(
			request.Key,
			"public/",
		) {
			return UploadTarget{},
				fmt.Errorf(
					"%w: public upload key must begin with public/",
					ErrInvalidKey,
				)
		}

	case AccessPrivate:
		if !strings.HasPrefix(
			request.Key,
			"private/",
		) {
			return UploadTarget{},
				fmt.Errorf(
					"%w: private upload key must begin with private/",
					ErrInvalidKey,
				)
		}

	default:
		return UploadTarget{},
			fmt.Errorf(
				"%w: invalid storage access %q",
				ErrInvalidKey,
				request.Access,
			)
	}

	return g.provider.CreateUpload(
		ctx,
		request,
	)
}

func (g *Gateway) CreateDownload(
	ctx context.Context,
	request DownloadRequest,
) (DownloadTarget, error) {
	request.Key = strings.TrimSpace(
		request.Key,
	)

	if request.Key == "" {
		return DownloadTarget{},
			ErrInvalidKey
	}

	return g.provider.CreateDownload(
		ctx,
		request,
	)
}

func (g *Gateway) PublicURL(
	key string,
) (string, error) {
	key = normalizeObjectKey(
		key,
	)

	if key == "" {
		return "",
			ErrInvalidKey
	}

	if !strings.HasPrefix(
		key,
		"public/",
	) {
		return "",
			fmt.Errorf(
				"%w: public URL requires a public/ object",
				ErrInvalidKey,
			)
	}

	return g.provider.PublicURL(
		key,
	)
}

func (g *Gateway) Stat(
	ctx context.Context,
	key string,
) (ObjectInfo, error) {
	key = strings.TrimSpace(key)

	if key == "" {
		return ObjectInfo{},
			ErrInvalidKey
	}

	return g.provider.Stat(
		ctx,
		key,
	)
}

func (g *Gateway) Delete(
	ctx context.Context,
	key string,
) error {
	key = strings.TrimSpace(key)

	if key == "" {
		return ErrInvalidKey
	}

	return g.provider.Delete(
		ctx,
		key,
	)
}
