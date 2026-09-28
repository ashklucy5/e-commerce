package storage

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	platformhttpclient "project.local/commerce-api/internal/platform/httpclient"
)

type s3HTTPProvider struct {
	*S3Provider

	client *platformhttpclient.Client
}

func newS3HTTPProvider(
	provider *S3Provider,
) Provider {
	cfg :=
		platformhttpclient.DefaultConfig()

	cfg.Timeout =
		15 * time.Second

	cfg.MaxResponseBytes =
		1 << 20

	return &s3HTTPProvider{
		S3Provider: provider,

		client: platformhttpclient.New(
			cfg,
		),
	}
}

func (p *s3HTTPProvider) Stat(
	ctx context.Context,
	key string,
) (ObjectInfo, error) {
	request, err :=
		p.signedRequest(
			ctx,
			http.MethodHead,
			key,
		)
	if err != nil {
		return ObjectInfo{}, err
	}

	response, err :=
		p.client.Do(
			request,
		)
	if err != nil {
		return ObjectInfo{},
			fmt.Errorf(
				"stat object: %w",
				err,
			)
	}

	defer response.Body.Close()

	if response.StatusCode ==
		http.StatusNotFound {

		return ObjectInfo{},
			ErrNotFound
	}

	if response.StatusCode <
		http.StatusOK ||
		response.StatusCode >=
			http.StatusMultipleChoices {

		return ObjectInfo{},
			fmt.Errorf(
				"stat object: storage returned %s",
				response.Status,
			)
	}

	contentLength :=
		response.ContentLength

	if contentLength < 0 {
		contentLength = 0
	}

	var lastModified time.Time

	lastModifiedHeader :=
		strings.TrimSpace(
			response.Header.Get(
				"Last-Modified",
			),
		)

	if lastModifiedHeader != "" {
		parsed,
			parseErr :=
			http.ParseTime(
				lastModifiedHeader,
			)

		if parseErr == nil {
			lastModified =
				parsed
		}
	}

	return ObjectInfo{
		Key: normalizeObjectKey(
			key,
		),

		ContentType: response.Header.Get(
			"Content-Type",
		),

		ContentLength: contentLength,

		ETag: strings.Trim(
			response.Header.Get(
				"ETag",
			),
			"\"",
		),

		LastModified: lastModified,
	}, nil
}

func (p *s3HTTPProvider) Delete(
	ctx context.Context,
	key string,
) error {
	request, err :=
		p.signedRequest(
			ctx,
			http.MethodDelete,
			key,
		)
	if err != nil {
		return err
	}

	response, err :=
		p.client.Do(
			request,
		)
	if err != nil {
		return fmt.Errorf(
			"delete object: %w",
			err,
		)
	}

	defer response.Body.Close()

	if response.StatusCode ==
		http.StatusNotFound {

		return ErrNotFound
	}

	if response.StatusCode <
		http.StatusOK ||
		response.StatusCode >=
			http.StatusMultipleChoices {

		return fmt.Errorf(
			"delete object: storage returned %s",
			response.Status,
		)
	}

	return nil
}
