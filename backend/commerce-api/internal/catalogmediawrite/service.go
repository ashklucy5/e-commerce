package catalogmediawrite

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"project.local/commerce-api/internal/platform/storage"
)

const (
	maxProductImageSize int64 = 15 * 1024 * 1024

	maxProduct360FrameSize int64 = 15 * 1024 * 1024

	maxProduct3DModelSize int64 = 50 * 1024 * 1024
)

var (
	ErrInvalidImageRequest = errors.New(
		"invalid product image request",
	)

	ErrInvalid360FrameRequest = errors.New(
		"invalid product 360 frame request",
	)

	ErrInvalid3DModelRequest = errors.New(
		"invalid product 3D model request",
	)

	ErrStorageUnavailable = errors.New(
		"storage is unavailable",
	)
)

type mediaRepository interface {
	CreateImage(
		ctx context.Context,
		input CreateImageInput,
		publicURL string,
	) (ImageResult, error)

	Upsert360Frame(
		ctx context.Context,
		input Create360FrameInput,
		publicURL string,
	) (Frame360Result, error)

	Upsert3DModel(
		ctx context.Context,
		input Upsert3DModelInput,
		publicURL string,
	) (Model3DResult, error)
}

type Service struct {
	repository mediaRepository
	storage    *storage.Gateway
}

func NewService(
	repository mediaRepository,
	storageGateway *storage.Gateway,
) *Service {
	if storageGateway == nil {
		storageGateway =
			storage.NewGateway(nil)
	}

	return &Service{
		repository: repository,
		storage:    storageGateway,
	}
}

func (s *Service) CreateImage(
	ctx context.Context,
	input CreateImageInput,
) (ImageResult, error) {
	input.ProductID =
		strings.TrimSpace(
			input.ProductID,
		)

	input.VariantID =
		strings.TrimSpace(
			input.VariantID,
		)

	input.StorageKey =
		strings.TrimSpace(
			input.StorageKey,
		)

	input.AltText =
		strings.TrimSpace(
			input.AltText,
		)

	if input.ProductID == "" {
		return ImageResult{},
			fmt.Errorf(
				"%w: product ID is required",
				ErrInvalidImageRequest,
			)
	}

	if input.StorageKey == "" {
		return ImageResult{},
			fmt.Errorf(
				"%w: storage_key is required",
				ErrInvalidImageRequest,
			)
	}

	if input.SortOrder < 0 {
		return ImageResult{},
			fmt.Errorf(
				"%w: sort_order cannot be negative",
				ErrInvalidImageRequest,
			)
	}

	if input.IsPrimary &&
		input.VariantID != "" {
		return ImageResult{},
			fmt.Errorf(
				"%w: a variant-specific image cannot be the product primary image",
				ErrInvalidImageRequest,
			)
	}

	if !strings.HasPrefix(
		input.StorageKey,
		"public/products/images/",
	) {
		return ImageResult{},
			fmt.Errorf(
				"%w: storage_key must reference a product image upload",
				ErrInvalidImageRequest,
			)
	}

	if s.storageDisabled() {
		return ImageResult{},
			ErrStorageUnavailable
	}

	info, err :=
		s.storage.Stat(
			ctx,
			input.StorageKey,
		)

	if err != nil {
		return ImageResult{},
			err
	}

	if info.ContentLength >
		maxProductImageSize {
		return ImageResult{},
			fmt.Errorf(
				"%w: uploaded product image exceeds 15 MiB limit",
				ErrInvalidImageRequest,
			)
	}

	contentType :=
		normalizedContentType(
			info.ContentType,
		)

	if contentType != "" &&
		!supportedImageContentType(
			contentType,
		) {
		return ImageResult{},
			fmt.Errorf(
				"%w: uploaded object has unsupported image content type %q",
				ErrInvalidImageRequest,
				contentType,
			)
	}

	publicURL, err :=
		s.storage.PublicURL(
			input.StorageKey,
		)

	if err != nil {
		return ImageResult{},
			err
	}

	return s.repository.CreateImage(
		ctx,
		input,
		publicURL,
	)
}

func (s *Service) Create360Frame(
	ctx context.Context,
	input Create360FrameInput,
) (Frame360Result, error) {
	input.ProductID =
		strings.TrimSpace(
			input.ProductID,
		)

	input.VariantID =
		strings.TrimSpace(
			input.VariantID,
		)

	input.StorageKey =
		strings.TrimSpace(
			input.StorageKey,
		)

	if input.ProductID == "" {
		return Frame360Result{},
			fmt.Errorf(
				"%w: product ID is required",
				ErrInvalid360FrameRequest,
			)
	}

	if input.StorageKey == "" {
		return Frame360Result{},
			fmt.Errorf(
				"%w: storage_key is required",
				ErrInvalid360FrameRequest,
			)
	}

	if input.FrameIndex < 0 {
		return Frame360Result{},
			fmt.Errorf(
				"%w: frame_index cannot be negative",
				ErrInvalid360FrameRequest,
			)
	}

	if !strings.HasPrefix(
		input.StorageKey,
		"public/products/360/",
	) {
		return Frame360Result{},
			fmt.Errorf(
				"%w: storage_key must reference a product 360 upload",
				ErrInvalid360FrameRequest,
			)
	}

	if s.storageDisabled() {
		return Frame360Result{},
			ErrStorageUnavailable
	}

	info, err :=
		s.storage.Stat(
			ctx,
			input.StorageKey,
		)

	if err != nil {
		return Frame360Result{},
			err
	}

	if info.ContentLength >
		maxProduct360FrameSize {
		return Frame360Result{},
			fmt.Errorf(
				"%w: uploaded 360 frame exceeds 15 MiB limit",
				ErrInvalid360FrameRequest,
			)
	}

	contentType :=
		normalizedContentType(
			info.ContentType,
		)

	if contentType != "" &&
		!supportedImageContentType(
			contentType,
		) {
		return Frame360Result{},
			fmt.Errorf(
				"%w: unsupported 360 frame content type %q",
				ErrInvalid360FrameRequest,
				contentType,
			)
	}

	publicURL, err :=
		s.storage.PublicURL(
			input.StorageKey,
		)

	if err != nil {
		return Frame360Result{},
			err
	}

	return s.repository.Upsert360Frame(
		ctx,
		input,
		publicURL,
	)
}

func (s *Service) Upsert3DModel(
	ctx context.Context,
	input Upsert3DModelInput,
) (Model3DResult, error) {
	input.ProductID =
		strings.TrimSpace(
			input.ProductID,
		)

	input.VariantID =
		strings.TrimSpace(
			input.VariantID,
		)

	input.StorageKey =
		strings.TrimSpace(
			input.StorageKey,
		)

	input.PosterURL =
		strings.TrimSpace(
			input.PosterURL,
		)

	if input.ProductID == "" {
		return Model3DResult{},
			fmt.Errorf(
				"%w: product ID is required",
				ErrInvalid3DModelRequest,
			)
	}

	if input.StorageKey == "" {
		return Model3DResult{},
			fmt.Errorf(
				"%w: storage_key is required",
				ErrInvalid3DModelRequest,
			)
	}

	if !strings.HasPrefix(
		input.StorageKey,
		"public/products/3d/",
	) {
		return Model3DResult{},
			fmt.Errorf(
				"%w: storage_key must reference a product 3D upload",
				ErrInvalid3DModelRequest,
			)
	}

	if input.PosterURL != "" &&
		!validHTTPURL(
			input.PosterURL,
		) {
		return Model3DResult{},
			fmt.Errorf(
				"%w: poster_url must be an http or https URL",
				ErrInvalid3DModelRequest,
			)
	}

	if s.storageDisabled() {
		return Model3DResult{},
			ErrStorageUnavailable
	}

	info, err :=
		s.storage.Stat(
			ctx,
			input.StorageKey,
		)

	if err != nil {
		return Model3DResult{},
			err
	}

	if info.ContentLength >
		maxProduct3DModelSize {
		return Model3DResult{},
			fmt.Errorf(
				"%w: uploaded 3D model exceeds 50 MiB limit",
				ErrInvalid3DModelRequest,
			)
	}

	contentType :=
		normalizedContentType(
			info.ContentType,
		)

	if contentType != "" &&
		!supported3DModelContentType(
			contentType,
		) {
		return Model3DResult{},
			fmt.Errorf(
				"%w: unsupported 3D model content type %q",
				ErrInvalid3DModelRequest,
				contentType,
			)
	}

	publicURL, err :=
		s.storage.PublicURL(
			input.StorageKey,
		)

	if err != nil {
		return Model3DResult{},
			err
	}

	return s.repository.Upsert3DModel(
		ctx,
		input,
		publicURL,
	)
}

func (s *Service) storageDisabled() bool {
	return strings.EqualFold(
		strings.TrimSpace(
			s.storage.ProviderName(),
		),
		"disabled",
	)
}

func normalizedContentType(
	value string,
) string {
	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	if index :=
		strings.Index(
			value,
			";",
		); index >= 0 {
		value =
			strings.TrimSpace(
				value[:index],
			)
	}

	return value
}

func supportedImageContentType(
	value string,
) bool {
	switch value {
	case "image/jpeg",
		"image/png",
		"image/webp",
		"image/avif":
		return true

	default:
		return false
	}
}

func supported3DModelContentType(
	value string,
) bool {
	switch value {
	case "model/gltf-binary",
		"application/octet-stream":
		return true

	default:
		return false
	}
}

func validHTTPURL(
	value string,
) bool {
	parsed, err :=
		url.ParseRequestURI(
			value,
		)

	if err != nil {
		return false
	}

	return parsed.Scheme == "http" ||
		parsed.Scheme == "https"
}
