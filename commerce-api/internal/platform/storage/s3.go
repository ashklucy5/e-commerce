package storage

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"project.local/commerce-api/internal/platform/config"
)

const (
	maxPresignedURLTTL = 7 * 24 * time.Hour
)

type S3Provider struct {
	endpoint *url.URL

	region string

	bucket string

	accessKey string

	secretKey string

	publicBaseURL string

	forcePathStyle bool

	uploadURLTTL time.Duration

	downloadURLTTL time.Duration

	httpClient *http.Client
}

func NewS3Provider(
	cfg config.StorageConfig,
) (*S3Provider, error) {
	endpointValue :=
		strings.TrimSpace(
			cfg.Endpoint,
		)

	if endpointValue == "" {
		return nil,
			fmt.Errorf(
				"storage endpoint is required",
			)
	}

	endpoint, err :=
		url.Parse(
			endpointValue,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"parse storage endpoint: %w",
				err,
			)
	}

	if endpoint.Scheme != "http" &&
		endpoint.Scheme != "https" {
		return nil,
			fmt.Errorf(
				"storage endpoint must use http or https",
			)
	}

	if strings.TrimSpace(
		endpoint.Host,
	) == "" {
		return nil,
			fmt.Errorf(
				"storage endpoint host is required",
			)
	}

	region :=
		strings.TrimSpace(
			cfg.Region,
		)

	if region == "" {
		region = "auto"
	}

	bucket :=
		strings.TrimSpace(
			cfg.Bucket,
		)

	if bucket == "" {
		return nil,
			fmt.Errorf(
				"storage bucket is required",
			)
	}

	accessKey :=
		strings.TrimSpace(
			cfg.AccessKey,
		)

	if accessKey == "" {
		return nil,
			fmt.Errorf(
				"storage access key is required",
			)
	}

	secretKey :=
		strings.TrimSpace(
			cfg.SecretKey,
		)

	if secretKey == "" {
		return nil,
			fmt.Errorf(
				"storage secret key is required",
			)
	}

	return &S3Provider{
		endpoint: endpoint,

		region: region,

		bucket: bucket,

		accessKey: accessKey,

		secretKey: secretKey,

		publicBaseURL: strings.TrimRight(
			strings.TrimSpace(
				cfg.PublicBaseURL,
			),
			"/",
		),

		forcePathStyle: cfg.ForcePathStyle,

		uploadURLTTL: normalizePresignTTL(
			cfg.UploadURLTTL,
		),

		downloadURLTTL: normalizePresignTTL(
			cfg.DownloadURLTTL,
		),

		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

func (p *S3Provider) Name() string {
	return "s3"
}

func (p *S3Provider) CreateUpload(
	_ context.Context,
	request UploadRequest,
) (UploadTarget, error) {
	key :=
		normalizeObjectKey(
			request.Key,
		)

	if key == "" {
		return UploadTarget{},
			ErrInvalidKey
	}

	expiresIn :=
		normalizePresignTTL(
			p.uploadURLTTL,
		)

	presignedURL,
		expiresAt,
		err :=
		p.presign(
			http.MethodPut,
			key,
			expiresIn,
		)

	if err != nil {
		return UploadTarget{},
			err
	}

	headers :=
		make(
			map[string]string,
		)

	if strings.TrimSpace(
		request.ContentType,
	) != "" {
		headers["Content-Type"] =
			strings.TrimSpace(
				request.ContentType,
			)
	}

	if strings.TrimSpace(
		request.CacheControl,
	) != "" {
		headers["Cache-Control"] =
			strings.TrimSpace(
				request.CacheControl,
			)
	}

	return UploadTarget{
		Provider: p.Name(),

		Key: key,

		Method: http.MethodPut,

		URL: presignedURL,

		Headers: headers,

		ExpiresAt: expiresAt,
	}, nil
}

func (p *S3Provider) CreateDownload(
	_ context.Context,
	request DownloadRequest,
) (DownloadTarget, error) {
	key :=
		normalizeObjectKey(
			request.Key,
		)

	if key == "" {
		return DownloadTarget{},
			ErrInvalidKey
	}

	expiresIn :=
		request.ExpiresIn

	if expiresIn <= 0 {
		expiresIn =
			p.downloadURLTTL
	}

	expiresIn =
		normalizePresignTTL(
			expiresIn,
		)

	presignedURL,
		expiresAt,
		err :=
		p.presign(
			http.MethodGet,
			key,
			expiresIn,
		)

	if err != nil {
		return DownloadTarget{},
			err
	}

	return DownloadTarget{
		Provider: p.Name(),

		Key: key,

		URL: presignedURL,

		ExpiresAt: expiresAt,
	}, nil
}

func (p *S3Provider) PublicURL(
	key string,
) (string, error) {
	key =
		normalizeObjectKey(
			key,
		)

	if key == "" {
		return "",
			ErrInvalidKey
	}

	const publicPrefix = "public/"

	if !strings.HasPrefix(
		key,
		publicPrefix,
	) {
		return "",
			fmt.Errorf(
				"%w: object is not public",
				ErrInvalidKey,
			)
	}

	if p.publicBaseURL != "" {
		publicKey :=
			strings.TrimPrefix(
				key,
				publicPrefix,
			)

		return p.publicBaseURL +
				"/" +
				escapeObjectKey(
					publicKey,
				),
			nil
	}

	objectURL, err :=
		p.objectURL(
			key,
		)

	if err != nil {
		return "", err
	}

	return objectURL.String(),
		nil
}

func (p *S3Provider) Stat(
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
		return ObjectInfo{},
			err
	}

	response, err :=
		p.httpClient.Do(
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

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
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
			lastModified = parsed
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

func (p *S3Provider) Delete(
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
		p.httpClient.Do(
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

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return fmt.Errorf(
			"delete object: storage returned %s",
			response.Status,
		)
	}

	return nil
}

func (p *S3Provider) presign(
	method string,
	key string,
	expiresIn time.Duration,
) (string, time.Time, error) {
	objectURL, err :=
		p.objectURL(
			key,
		)

	if err != nil {
		return "",
			time.Time{},
			err
	}

	now :=
		time.Now().
			UTC()

	expiresIn =
		normalizePresignTTL(
			expiresIn,
		)

	expiresAt :=
		now.Add(
			expiresIn,
		)

	scope :=
		credentialScope(
			now,
			p.region,
		)

	query :=
		objectURL.Query()

	query.Set(
		"X-Amz-Algorithm",
		awsAlgorithm,
	)

	query.Set(
		"X-Amz-Credential",
		p.accessKey+
			"/"+
			scope,
	)

	query.Set(
		"X-Amz-Date",
		now.Format(
			"20060102T150405Z",
		),
	)

	query.Set(
		"X-Amz-Expires",
		strconv.FormatInt(
			int64(
				expiresIn/
					time.Second,
			),
			10,
		),
	)

	query.Set(
		"X-Amz-SignedHeaders",
		"host",
	)

	objectURL.RawQuery =
		canonicalQuery(
			query,
		)

	canonicalRequest :=
		strings.Join(
			[]string{
				method,

				objectURL.EscapedPath(),

				objectURL.RawQuery,

				"host:" +
					strings.ToLower(
						objectURL.Host,
					) +
					"\n",

				"host",

				unsignedPayload,
			},
			"\n",
		)

	stringToSign :=
		strings.Join(
			[]string{
				awsAlgorithm,

				now.Format(
					"20060102T150405Z",
				),

				scope,

				sha256Hex(
					canonicalRequest,
				),
			},
			"\n",
		)

	signature :=
		hmacSHA256(
			signingKey(
				p.secretKey,

				now.Format(
					"20060102",
				),

				p.region,
			),
			stringToSign,
		)

	query.Set(
		"X-Amz-Signature",
		hex.EncodeToString(
			signature,
		),
	)

	objectURL.RawQuery =
		canonicalQuery(
			query,
		)

	return objectURL.String(),
		expiresAt,
		nil
}

func (p *S3Provider) signedRequest(
	ctx context.Context,
	method string,
	key string,
) (*http.Request, error) {
	objectURL, err :=
		p.objectURL(
			key,
		)

	if err != nil {
		return nil, err
	}

	now :=
		time.Now().
			UTC()

	payloadHash :=
		sha256Hex("")

	scope :=
		credentialScope(
			now,
			p.region,
		)

	canonicalHeaders :=
		strings.Join(
			[]string{
				"host:" +
					strings.ToLower(
						objectURL.Host,
					),

				"x-amz-content-sha256:" +
					payloadHash,

				"x-amz-date:" +
					now.Format(
						"20060102T150405Z",
					),

				"",
			},
			"\n",
		)

	signedHeaders :=
		"host;x-amz-content-sha256;x-amz-date"

	canonicalRequest :=
		strings.Join(
			[]string{
				method,

				objectURL.EscapedPath(),

				canonicalQuery(
					objectURL.Query(),
				),

				canonicalHeaders,

				signedHeaders,

				payloadHash,
			},
			"\n",
		)

	stringToSign :=
		strings.Join(
			[]string{
				awsAlgorithm,

				now.Format(
					"20060102T150405Z",
				),

				scope,

				sha256Hex(
					canonicalRequest,
				),
			},
			"\n",
		)

	signature :=
		hex.EncodeToString(
			hmacSHA256(
				signingKey(
					p.secretKey,

					now.Format(
						"20060102",
					),

					p.region,
				),
				stringToSign,
			),
		)

	request,
		err :=
		http.NewRequestWithContext(
			ctx,
			method,
			objectURL.String(),
			nil,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"create storage request: %w",
				err,
			)
	}

	request.Header.Set(
		"X-Amz-Date",
		now.Format(
			"20060102T150405Z",
		),
	)

	request.Header.Set(
		"X-Amz-Content-Sha256",
		payloadHash,
	)

	request.Header.Set(
		"Authorization",
		fmt.Sprintf(
			"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
			awsAlgorithm,
			p.accessKey,
			scope,
			signedHeaders,
			signature,
		),
	)

	return request,
		nil
}

func (p *S3Provider) objectURL(
	key string,
) (*url.URL, error) {
	key =
		normalizeObjectKey(
			key,
		)

	if key == "" {
		return nil,
			ErrInvalidKey
	}

	result := *p.endpoint

	if p.forcePathStyle {
		result.Path =
			path.Join(
				result.Path,
				p.bucket,
				key,
			)
	} else {
		result.Host =
			p.bucket +
				"." +
				result.Host

		result.Path =
			path.Join(
				result.Path,
				key,
			)
	}

	if !strings.HasPrefix(
		result.Path,
		"/",
	) {
		result.Path =
			"/" +
				result.Path
	}

	return &result,
		nil
}

func normalizeObjectKey(
	key string,
) string {
	key =
		strings.TrimSpace(
			key,
		)

	key =
		strings.ReplaceAll(
			key,
			"\\",
			"/",
		)

	key =
		strings.TrimLeft(
			key,
			"/",
		)

	if key == "" {
		return ""
	}

	segments :=
		strings.Split(
			key,
			"/",
		)

	clean :=
		make(
			[]string,
			0,
			len(segments),
		)

	for _, segment := range segments {
		segment =
			strings.TrimSpace(
				segment,
			)

		if segment == "" ||
			segment == "." {
			continue
		}

		if segment == ".." {
			return ""
		}

		clean = append(
			clean,
			segment,
		)
	}

	if len(clean) == 0 {
		return ""
	}

	return strings.Join(
		clean,
		"/",
	)
}

func escapeObjectKey(
	key string,
) string {
	key =
		normalizeObjectKey(
			key,
		)

	if key == "" {
		return ""
	}

	segments :=
		strings.Split(
			key,
			"/",
		)

	for index := range segments {
		segments[index] =
			url.PathEscape(
				segments[index],
			)
	}

	return strings.Join(
		segments,
		"/",
	)
}

func normalizePresignTTL(
	value time.Duration,
) time.Duration {
	if value <= 0 {
		return 15 *
			time.Minute
	}

	if value >
		maxPresignedURLTTL {
		return maxPresignedURLTTL
	}

	if value <
		time.Second {
		return time.Second
	}

	return value
}
