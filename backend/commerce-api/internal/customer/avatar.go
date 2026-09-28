package customer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"

	"project.local/commerce-api/internal/platform/storage"
)

const (
	maxAvatarBytes int64 = 5 * 1024 * 1024
)

var (
	ErrInvalidAvatarContentType = errors.New(
		"avatar must be JPEG, PNG, or WebP",
	)

	ErrInvalidAvatarSize = errors.New(
		"avatar must be between 1 byte and 5 MB",
	)

	ErrInvalidAvatarKey = errors.New(
		"invalid avatar storage key",
	)

	ErrAvatarUploadNotFound = errors.New(
		"uploaded avatar was not found",
	)
)

type AvatarUploadTargetRequest struct {
	ContentType   string `json:"content_type"`
	ContentLength int64  `json:"content_length"`
}

type AvatarCompleteRequest struct {
	Key string `json:"key"`
}

type AvatarUploadTarget struct {
	storage.UploadTarget
}

func (s *Service) CreateAvatarUploadTarget(
	ctx context.Context,
	customerID string,
	request AvatarUploadTargetRequest,
) (AvatarUploadTarget, error) {
	contentType :=
		normalizeAvatarContentType(
			request.ContentType,
		)

	extension, ok :=
		avatarExtension(
			contentType,
		)
	if !ok {
		return AvatarUploadTarget{},
			ErrInvalidAvatarContentType
	}

	if request.ContentLength <= 0 ||
		request.ContentLength >
			maxAvatarBytes {
		return AvatarUploadTarget{},
			ErrInvalidAvatarSize
	}

	// Also proves the customer exists before issuing
	// an upload target.
	if _, err :=
		s.repository.GetCustomer(
			ctx,
			customerID,
		); err != nil {
		return AvatarUploadTarget{},
			err
	}

	randomID, err :=
		randomAvatarID()
	if err != nil {
		return AvatarUploadTarget{},
			fmt.Errorf(
				"generate avatar object id: %w",
				err,
			)
	}

	key :=
		fmt.Sprintf(
			"private/avatars/%s/%s%s",
			customerID,
			randomID,
			extension,
		)

	target, err :=
		s.storage.CreateUpload(
			ctx,
			storage.UploadRequest{
				Key: key,

				ContentType: contentType,

				ContentLength: request.ContentLength,

				Access: storage.AccessPrivate,

				Purpose: storage.PurposeCustomerAvatar,

				CacheControl: "private, max-age=3600",
			},
		)
	if err != nil {
		return AvatarUploadTarget{},
			err
	}

	return AvatarUploadTarget{
		UploadTarget: target,
	}, nil
}

func (s *Service) CompleteAvatarUpload(
	ctx context.Context,
	customerID string,
	request AvatarCompleteRequest,
) (Customer, error) {
	key :=
		strings.TrimSpace(
			request.Key,
		)

	if !validCustomerAvatarKey(
		customerID,
		key,
	) {
		return Customer{},
			ErrInvalidAvatarKey
	}

	objectInfo, err :=
		s.storage.Stat(
			ctx,
			key,
		)
	if err != nil {
		if errors.Is(
			err,
			storage.ErrNotFound,
		) {
			return Customer{},
				ErrAvatarUploadNotFound
		}

		return Customer{},
			err
	}

	if objectInfo.ContentLength <= 0 ||
		objectInfo.ContentLength >
			maxAvatarBytes {
		return Customer{},
			ErrInvalidAvatarSize
	}

	contentType :=
		normalizeAvatarContentType(
			objectInfo.ContentType,
		)

	extension, ok :=
		avatarExtension(
			contentType,
		)
	if !ok {
		return Customer{},
			ErrInvalidAvatarContentType
	}

	if !strings.EqualFold(
		path.Ext(key),
		extension,
	) {
		return Customer{},
			ErrInvalidAvatarContentType
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return Customer{},
			err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	previousStorageKey,
		_,
		err :=
		s.repository.SetAvatarTx(
			ctx,
			tx,
			customerID,
			key,
		)
	if err != nil {
		return Customer{},
			err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return Customer{},
			fmt.Errorf(
				"commit customer avatar update: %w",
				err,
			)
	}

	if previousStorageKey != "" &&
		previousStorageKey != key {
		_ = s.storage.Delete(
			ctx,
			previousStorageKey,
		)
	}

	return s.GetProfile(
		ctx,
		customerID,
	)
}

func (s *Service) DeleteAvatar(
	ctx context.Context,
	customerID string,
) error {
	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	previousStorageKey, err :=
		s.repository.ClearAvatarTx(
			ctx,
			tx,
			customerID,
		)
	if err != nil {
		return err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return fmt.Errorf(
			"commit customer avatar deletion: %w",
			err,
		)
	}

	if previousStorageKey != "" {
		_ = s.storage.Delete(
			ctx,
			previousStorageKey,
		)
	}

	return nil
}

func (s *Service) attachAvatar(
	ctx context.Context,
	customerID string,
	result *Customer,
) error {
	state, err :=
		s.repository.GetAvatarState(
			ctx,
			customerID,
		)
	if err != nil {
		return err
	}

	result.AvatarUpdatedAt =
		state.UpdatedAt

	if state.StorageKey == "" {
		return nil
	}

	target, err :=
		s.storage.CreateDownload(
			ctx,
			storage.DownloadRequest{
				Key: state.StorageKey,
			},
		)
	if err != nil {
		if errors.Is(
			err,
			storage.ErrDisabled,
		) {
			return nil
		}

		return err
	}

	result.AvatarURL =
		target.URL

	expiresAt :=
		target.ExpiresAt

	result.AvatarURLExpiresAt =
		&expiresAt

	return nil
}

func normalizeAvatarContentType(
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

func avatarExtension(
	contentType string,
) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true

	case "image/png":
		return ".png", true

	case "image/webp":
		return ".webp", true

	default:
		return "", false
	}
}

func validCustomerAvatarKey(
	customerID string,
	key string,
) bool {
	prefix :=
		"private/avatars/" +
			customerID +
			"/"

	if !strings.HasPrefix(
		key,
		prefix,
	) {
		return false
	}

	fileName :=
		strings.TrimPrefix(
			key,
			prefix,
		)

	if fileName == "" ||
		strings.Contains(
			fileName,
			"/",
		) ||
		strings.Contains(
			fileName,
			"..",
		) {
		return false
	}

	switch strings.ToLower(
		path.Ext(fileName),
	) {
	case ".jpg",
		".png",
		".webp":
		return true

	default:
		return false
	}
}

func randomAvatarID() (
	string,
	error,
) {
	buffer :=
		make(
			[]byte,
			16,
		)

	if _, err :=
		rand.Read(
			buffer,
		); err != nil {
		return "", err
	}

	return hex.EncodeToString(
		buffer,
	), nil
}
