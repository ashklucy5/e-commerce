package staff

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Service struct {
	repository *Repository
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Login(
	ctx context.Context,
	request LoginRequest,
) (AuthResult, error) {
	identifier :=
		strings.TrimSpace(
			request.Identifier,
		)

	// Passwords are opaque values.
	//
	// Do not TrimSpace here. Leading/trailing spaces may legitimately
	// be part of a password and changing the supplied value would make
	// verification inconsistent with what was originally hashed.
	password :=
		request.Password

	if identifier == "" ||
		password == "" {
		return AuthResult{},
			ErrInvalidCredentials
	}

	record, err :=
		s.repository.FindByIdentifier(
			ctx,
			identifier,
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return AuthResult{},
				ErrInvalidCredentials
		}

		return AuthResult{}, err
	}

	if record.Status !=
		"active" {
		return AuthResult{},
			ErrStaffDisabled
	}

	valid,
		replacementPasswordHash,
		err :=
		verifyPassword(
			password,
			record.PasswordHash,
		)
	if err != nil {
		return AuthResult{}, err
	}

	if !valid {
		return AuthResult{},
			ErrInvalidCredentials
	}

	material, err :=
		newTokenMaterial(
			time.Now().UTC(),
		)
	if err != nil {
		return AuthResult{}, err
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return AuthResult{}, err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	// Legacy bcrypt staff passwords are transparently migrated to
	// Argon2id only after the existing password has been successfully
	// verified.
	//
	// Current Argon2id hashes are also transparently upgraded when the
	// configured work factor becomes stronger in the future.
	if replacementPasswordHash != "" {
		if err :=
			s.repository.UpdatePasswordHashTx(
				ctx,
				tx,
				record.ID,
				replacementPasswordHash,
			); err != nil {
			return AuthResult{}, err
		}
	}

	if err :=
		s.repository.CreateSessionTx(
			ctx,
			tx,
			record.ID,
			material,
		); err != nil {
		return AuthResult{}, err
	}

	if err :=
		s.repository.UpdateLastLoginTx(
			ctx,
			tx,
			record.ID,
		); err != nil {
		return AuthResult{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return AuthResult{}, err
	}

	account, err :=
		s.repository.GetAccountByID(
			ctx,
			record.ID,
		)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		Staff: account,

		Tokens: tokensFromMaterial(
			material,
		),
	}, nil
}

func (s *Service) Refresh(
	ctx context.Context,
	request RefreshRequest,
) (AuthResult, error) {
	refreshToken :=
		strings.TrimSpace(
			request.RefreshToken,
		)

	if refreshToken == "" {
		return AuthResult{},
			ErrInvalidRefreshToken
	}

	tx, err :=
		s.repository.Begin(
			ctx,
		)
	if err != nil {
		return AuthResult{}, err
	}

	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	session, err :=
		s.repository.LockSessionByRefreshTokenTx(
			ctx,
			tx,
			hashToken(
				refreshToken,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return AuthResult{},
				ErrInvalidRefreshToken
		}

		return AuthResult{}, err
	}

	now :=
		time.Now().UTC()

	if !session.RefreshExpiresAt.
		After(
			now,
		) {
		return AuthResult{},
			ErrInvalidRefreshToken
	}

	if session.StaffStatus !=
		"active" {
		return AuthResult{},
			ErrStaffDisabled
	}

	material, err :=
		newTokenMaterial(
			now,
		)
	if err != nil {
		return AuthResult{}, err
	}

	if err :=
		s.repository.RotateSessionTx(
			ctx,
			tx,
			session.ID,
			material,
		); err != nil {
		return AuthResult{}, err
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return AuthResult{}, err
	}

	account, err :=
		s.repository.GetAccountByID(
			ctx,
			session.StaffAccountID,
		)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		Staff: account,

		Tokens: tokensFromMaterial(
			material,
		),
	}, nil
}

func (s *Service) Logout(
	ctx context.Context,
	accessToken string,
) error {
	accessToken =
		strings.TrimSpace(
			accessToken,
		)

	if accessToken == "" {
		return ErrInvalidAccessToken
	}

	return s.repository.
		RevokeSessionByAccessTokenHash(
			ctx,
			hashToken(
				accessToken,
			),
		)
}

func (s *Service) AuthenticateAccessToken(
	ctx context.Context,
	accessToken string,
) (Account, error) {
	accessToken =
		strings.TrimSpace(
			accessToken,
		)

	if accessToken == "" {
		return Account{},
			ErrInvalidAccessToken
	}

	account,
		accessExpiresAt,
		err :=
		s.repository.GetAccountByAccessTokenHash(
			ctx,
			hashToken(
				accessToken,
			),
		)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return Account{},
				ErrInvalidAccessToken
		}

		return Account{}, err
	}

	if !accessExpiresAt.After(
		time.Now().UTC(),
	) {
		return Account{},
			ErrInvalidAccessToken
	}

	if account.Status !=
		"active" {
		return Account{},
			ErrStaffDisabled
	}

	return account, nil
}
