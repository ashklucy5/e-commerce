package adminupload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"project.local/commerce-api/internal/platform/storage"
)

const (
	maxProductImageSize int64 = 15 * 1024 * 1024

	maxProduct360FrameSize int64 = 15 * 1024 * 1024

	maxProduct3DModelSize int64 = 50 * 1024 * 1024

	maxImportAssetSize int64 = 20 * 1024 * 1024
)

var (
	ErrInvalidUploadRequest = errors.New(
		"invalid upload request",
	)

	ErrUnsupportedPurpose = errors.New(
		"unsupported upload purpose",
	)

	ErrUnsupportedFileType = errors.New(
		"unsupported file type",
	)

	ErrFileTooLarge = errors.New(
		"file too large",
	)

	ErrStorageUnavailable = errors.New(
		"storage is unavailable",
	)
)

type Service struct {
	storage *storage.Gateway
}

func NewService(
	storageGateway *storage.Gateway,
) *Service {
	return &Service{
		storage: storageGateway,
	}
}

func (s *Service) CreateUpload(
	ctx context.Context,
	request CreateUploadRequest,
) (*CreateUploadResponse, error) {
	if s.storage == nil {
		return nil,
			ErrStorageUnavailable
	}

	if strings.EqualFold(
		strings.TrimSpace(
			s.storage.ProviderName(),
		),
		"disabled",
	) {
		return nil,
			ErrStorageUnavailable
	}

	filename :=
		normalizeFilename(
			request.Filename,
		)

	if filename == "" {
		return nil,
			fmt.Errorf(
				"%w: filename is required",
				ErrInvalidUploadRequest,
			)
	}

	contentType :=
		normalizeContentType(
			request.ContentType,
		)

	if contentType == "" {
		return nil,
			fmt.Errorf(
				"%w: content_type is required",
				ErrInvalidUploadRequest,
			)
	}

	if request.ContentLength <= 0 {
		return nil,
			fmt.Errorf(
				"%w: content_length must be greater than zero",
				ErrInvalidUploadRequest,
			)
	}

	spec, err :=
		validateUpload(
			filename,
			contentType,
			request.ContentLength,
			request.Purpose,
		)

	if err != nil {
		return nil,
			err
	}

	key, err :=
		createStorageKey(
			spec.prefix,
			filename,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"create storage key: %w",
				err,
			)
	}

	target, err :=
		s.storage.CreateUpload(
			ctx,
			storage.UploadRequest{
				Key: key,

				ContentType: contentType,

				ContentLength: request.ContentLength,

				Access: spec.access,

				Purpose: request.Purpose,

				CacheControl: spec.cacheControl,
			},
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"create upload target: %w",
				err,
			)
	}

	return &CreateUploadResponse{
		Filename: filename,

		Purpose: request.Purpose,

		Upload: target,
	}, nil
}

type uploadSpec struct {
	prefix string

	access storage.Access

	cacheControl string
}

func validateUpload(
	filename string,
	contentType string,
	contentLength int64,
	purpose storage.Purpose,
) (uploadSpec, error) {
	switch purpose {

	case storage.PurposeProductImage:
		if contentLength >
			maxProductImageSize {
			return uploadSpec{},
				fmt.Errorf(
					"%w: product images cannot exceed 15 MiB",
					ErrFileTooLarge,
				)
		}

		if !validProductImage(
			filename,
			contentType,
		) {
			return uploadSpec{},
				fmt.Errorf(
					"%w: product images must be JPG, JPEG, PNG, WEBP, or AVIF",
					ErrUnsupportedFileType,
				)
		}

		return uploadSpec{
			prefix: "public/products/images/uploads",

			access: storage.AccessPublic,

			cacheControl: "public, max-age=31536000, immutable",
		}, nil

	case storage.PurposeProduct360Frame:
		if contentLength >
			maxProduct360FrameSize {
			return uploadSpec{},
				fmt.Errorf(
					"%w: product 360 frames cannot exceed 15 MiB",
					ErrFileTooLarge,
				)
		}

		if !validProductImage(
			filename,
			contentType,
		) {
			return uploadSpec{},
				fmt.Errorf(
					"%w: product 360 frames must be JPG, JPEG, PNG, WEBP, or AVIF",
					ErrUnsupportedFileType,
				)
		}

		return uploadSpec{
			prefix: "public/products/360/uploads",

			access: storage.AccessPublic,

			cacheControl: "public, max-age=31536000, immutable",
		}, nil

	case storage.PurposeProduct3DModel:
		if contentLength >
			maxProduct3DModelSize {
			return uploadSpec{},
				fmt.Errorf(
					"%w: product 3D models cannot exceed 50 MiB",
					ErrFileTooLarge,
				)
		}

		if !validProduct3DModel(
			filename,
			contentType,
		) {
			return uploadSpec{},
				fmt.Errorf(
					"%w: product 3D models must be GLB",
					ErrUnsupportedFileType,
				)
		}

		return uploadSpec{
			prefix: "public/products/3d/uploads",

			access: storage.AccessPublic,

			cacheControl: "public, max-age=31536000, immutable",
		}, nil

	case storage.PurposeImportAsset:
		if contentLength >
			maxImportAssetSize {
			return uploadSpec{},
				fmt.Errorf(
					"%w: catalog import files cannot exceed 20 MiB",
					ErrFileTooLarge,
				)
		}

		if !validImportAsset(
			filename,
			contentType,
		) {
			return uploadSpec{},
				fmt.Errorf(
					"%w: catalog import files must be XLSX",
					ErrUnsupportedFileType,
				)
		}

		return uploadSpec{
			prefix: "private/catalog-imports/uploads",

			access: storage.AccessPrivate,
		}, nil

	default:
		return uploadSpec{},
			fmt.Errorf(
				"%w: %q",
				ErrUnsupportedPurpose,
				purpose,
			)
	}
}

func validProductImage(
	filename string,
	contentType string,
) bool {
	extension :=
		strings.ToLower(
			path.Ext(
				filename,
			),
		)

	switch extension {

	case ".jpg",
		".jpeg":
		return contentType ==
			"image/jpeg"

	case ".png":
		return contentType ==
			"image/png"

	case ".webp":
		return contentType ==
			"image/webp"

	case ".avif":
		return contentType ==
			"image/avif"

	default:
		return false
	}
}

func validProduct3DModel(
	filename string,
	contentType string,
) bool {
	extension :=
		strings.ToLower(
			path.Ext(
				filename,
			),
		)

	if extension != ".glb" {
		return false
	}

	switch contentType {

	case "model/gltf-binary":
		return true

	case "application/octet-stream":
		return true

	default:
		return false
	}
}

func validImportAsset(
	filename string,
	contentType string,
) bool {
	if strings.ToLower(
		path.Ext(
			filename,
		),
	) != ".xlsx" {
		return false
	}

	switch contentType {

	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return true

	case "application/octet-stream":
		return true

	default:
		return false
	}
}

func normalizeContentType(
	value string,
) string {
	value =
		strings.TrimSpace(
			strings.ToLower(
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

func normalizeFilename(
	value string,
) string {
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

	if value == "." ||
		value == "/" ||
		value == "" {

		return ""
	}

	extension :=
		strings.ToLower(
			path.Ext(
				value,
			),
		)

	name :=
		strings.TrimSuffix(
			value,
			path.Ext(
				value,
			),
		)

	name =
		sanitizeFilenamePart(
			name,
		)

	if name == "" {
		name = "upload"
	}

	return name +
		extension
}

func sanitizeFilenamePart(
	value string,
) string {
	var builder strings.Builder

	lastSeparator :=
		false

	for _, character := range value {

		valid :=
			character >= 'a' &&
				character <= 'z' ||
				character >= 'A' &&
					character <= 'Z' ||
				character >= '0' &&
					character <= '9' ||
				character == '-' ||
				character == '_'

		if valid {
			builder.WriteRune(
				character,
			)

			lastSeparator =
				false

			continue
		}

		if !lastSeparator {
			builder.WriteByte(
				'-',
			)

			lastSeparator =
				true
		}
	}

	return strings.Trim(
		builder.String(),
		"-_",
	)
}

func createStorageKey(
	prefix string,
	filename string,
) (string, error) {
	randomID, err :=
		randomHex(
			16,
		)

	if err != nil {
		return "",
			err
	}

	now :=
		time.Now().
			UTC()

	return fmt.Sprintf(
		"%s/%04d/%02d/%s-%s",
		strings.Trim(
			prefix,
			"/",
		),
		now.Year(),
		int(
			now.Month(),
		),
		randomID,
		filename,
	), nil
}

func randomHex(
	byteLength int,
) (string, error) {
	if byteLength <= 0 {
		return "",
			fmt.Errorf(
				"byte length must be greater than zero",
			)
	}

	buffer :=
		make(
			[]byte,
			byteLength,
		)

	if _, err :=
		rand.Read(
			buffer,
		); err != nil {

		return "",
			fmt.Errorf(
				"generate random identifier: %w",
				err,
			)
	}

	return hex.EncodeToString(
		buffer,
	), nil
}
