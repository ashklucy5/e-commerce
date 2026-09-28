package adminauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	AdminMFARotationTTL = 10 * time.Minute

	SecurityEventMFARotationStarted       = "mfa_rotation_started"
	SecurityEventMFARotationCompleted     = "mfa_rotation_completed"
	SecurityEventRecoveryCodesRegenerated = "recovery_codes_regenerated"
)

var (
	ErrInvalidMFARotation = errors.New(
		"invalid or expired Admin MFA rotation",
	)
)

type SecurityStepUpRequest struct {
	Password string `json:"password"`

	Method string `json:"method"`

	Code string `json:"code"`
}

type BeginMFARotationRequest struct {
	SecurityStepUpRequest

	Label string `json:"label"`
}

type BeginMFARotationResponse struct {
	RotationToken string `json:"rotation_token"`

	RotationExpiresAt time.Time `json:"rotation_expires_at"`

	Enrollment MFAEnrollment `json:"enrollment"`
}

type ConfirmMFARotationRequest struct {
	RotationToken string `json:"rotation_token"`

	Code string `json:"code"`
}

type RegenerateRecoveryCodesRequest struct {
	SecurityStepUpRequest
}

type SelfSecurityMutationResponse struct {
	Principal AdminPrincipal `json:"principal"`

	CSRFToken string `json:"csrf_token"`

	RecoveryCodes []string `json:"recovery_codes"`
}

type SelfSecurityMutationResult struct {
	Session SessionResult

	RecoveryCodes []string
}

type pendingMFARotation struct {
	StaffAccountID string `json:"staff_account_id"`

	SessionID string `json:"session_id"`

	RotationTokenHash string `json:"rotation_token_hash"`

	CredentialID string `json:"credential_id"`

	Label string `json:"label"`

	SecretCiphertext string `json:"secret_ciphertext"`

	EncryptionKeyID string `json:"encryption_key_id"`

	Algorithm string `json:"algorithm"`

	Digits int `json:"digits"`

	PeriodSeconds int `json:"period_seconds"`

	CreatedAt time.Time `json:"created_at"`

	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Service) BeginSelfMFARotation(
	ctx context.Context,
	accessToken string,
	csrfToken string,
	request BeginMFARotationRequest,
	metadata ClientMetadata,
) (
	BeginMFARotationResponse,
	error,
) {
	accessToken = strings.TrimSpace(accessToken)
	csrfToken = strings.TrimSpace(csrfToken)
	request.Method = strings.ToLower(strings.TrimSpace(request.Method))
	request.Code = strings.TrimSpace(request.Code)
	request.Label = strings.TrimSpace(request.Label)

	if accessToken == "" {
		return BeginMFARotationResponse{}, ErrInvalidAccessToken
	}

	if csrfToken == "" {
		return BeginMFARotationResponse{}, ErrInvalidCSRFToken
	}

	if request.Password == "" || request.Code == "" || !validSelfSecurityMFAMethod(request.Method) {
		return BeginMFARotationResponse{}, ErrInvalidRequest
	}

	if request.Label == "" {
		request.Label = "Authenticator"
	}
	if len([]rune(request.Label)) > 120 {
		return BeginMFARotationResponse{}, ErrInvalidRequest
	}

	principal, err := s.AuthenticateAccessTokenWithCSRF(ctx, accessToken, csrfToken)
	if err != nil {
		return BeginMFARotationResponse{}, err
	}

	blocked, err := s.lockout.IsMFABlocked(ctx, principal.Staff.ID, metadata.IPAddress)
	if err != nil {
		return BeginMFARotationResponse{}, fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	if blocked {
		return BeginMFARotationResponse{}, ErrMFABlocked
	}

	rotationToken, err := platformsecurity.RandomURLSafe(32)
	if err != nil {
		return BeginMFARotationResponse{}, fmt.Errorf("generate Admin MFA rotation token: %w", err)
	}

	newSecret, err := GenerateTOTPSecret()
	if err != nil {
		return BeginMFARotationResponse{}, err
	}

	now := s.now().UTC()
	expiresAt := now.Add(AdminMFARotationTTL)

	var pending pendingMFARotation
	var response BeginMFARotationResponse
	var publicErr error
	var stepUpFailed bool
	staffIDForLockout := principal.Staff.ID

	err = platformdatabase.WithinTx(
		ctx,
		s.repository.db,
		func(ctx context.Context, tx pgx.Tx) error {
			session, err := s.repository.LockSelfSecuritySessionByAccessTokenHashTx(
				ctx,
				tx,
				platformsecurity.HashToken(accessToken),
			)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					publicErr = ErrInvalidAccessToken
					return nil
				}

				return err
			}

			if err := validateSelfSecuritySession(session, csrfToken, now); err != nil {
				publicErr = err
				return nil
			}
			if session.ID != principal.SessionID || session.StaffAccountID != principal.Staff.ID {
				publicErr = ErrInvalidAccessToken
				return nil
			}

			validPassword, replacementHash, err := verifyStoredPassword(request.Password, session.PasswordHash)
			if err != nil {
				return err
			}
			if !validPassword {
				stepUpFailed = true
				publicErr = ErrInvalidCredentials
				return nil
			}

			credential, err := s.repository.LockTOTPCredentialTx(ctx, tx, session.StaffAccountID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					publicErr = ErrMFANotEnrolled
					return nil
				}
				return err
			}
			if credential.Status != TOTPCredentialStatusActive {
				publicErr = ErrMFANotEnrolled
				return nil
			}

			usedRecoveryCode, valid, err := s.verifySelfSecurityStepUpTx(
				ctx,
				tx,
				session,
				credential,
				request.Method,
				request.Code,
				metadata,
				now,
			)
			if err != nil {
				return err
			}
			if !valid {
				stepUpFailed = true
				publicErr = ErrInvalidMFACode
				return nil
			}

			if replacementHash != "" {
				if err := s.repository.UpdatePasswordHashTx(ctx, tx, session.StaffAccountID, replacementHash); err != nil {
					return err
				}
			}

			cfg := totpConfigFromCredential(credential)
			encrypted, err := s.mfa.keyring.EncryptString(
				newSecret,
				totpAAD(session.StaffAccountID, credential.ID),
			)
			if err != nil {
				return fmt.Errorf("encrypt pending Admin MFA rotation secret: %w", err)
			}

			pending = pendingMFARotation{
				StaffAccountID:    session.StaffAccountID,
				SessionID:         session.ID,
				RotationTokenHash: platformsecurity.HashToken(rotationToken),
				CredentialID:      credential.ID,
				Label:             request.Label,
				SecretCiphertext:  encrypted.Ciphertext,
				EncryptionKeyID:   encrypted.KeyID,
				Algorithm:         cfg.Algorithm,
				Digits:            cfg.Digits,
				PeriodSeconds:     int(cfg.PeriodSeconds),
				CreatedAt:         now,
				ExpiresAt:         expiresAt,
			}

			staffID := session.StaffAccountID
			sessionID := session.ID
			if err := s.repository.InsertSecurityEventTx(
				ctx,
				tx,
				newSecurityEvent(
					SecurityEventMFARotationStarted,
					SecurityOutcomeSuccess,
					&staffID,
					&sessionID,
					"",
					metadata,
					map[string]any{
						"method":             request.Method,
						"credential_id":      credential.ID,
						"used_recovery_code": usedRecoveryCode,
						"expires_at":         expiresAt,
					},
					now,
				),
			); err != nil {
				return err
			}

			enrollmentURI, err := BuildTOTPEnrollmentURI(
				s.mfa.issuer,
				session.Email,
				newSecret,
				cfg,
			)
			if err != nil {
				return err
			}

			response = BeginMFARotationResponse{
				RotationToken:     rotationToken,
				RotationExpiresAt: expiresAt,
				Enrollment: MFAEnrollment{
					CredentialID:  credential.ID,
					Label:         request.Label,
					Secret:        newSecret,
					EnrollmentURI: enrollmentURI,
					Algorithm:     cfg.Algorithm,
					Digits:        cfg.Digits,
					PeriodSeconds: int(cfg.PeriodSeconds),
				},
			}

			return nil
		},
	)
	if err != nil {
		return BeginMFARotationResponse{}, err
	}

	if stepUpFailed && staffIDForLockout != "" {
		_ = s.lockout.RecordMFAFailure(ctx, staffIDForLockout, metadata.IPAddress)
	}

	if publicErr != nil {
		return BeginMFARotationResponse{}, publicErr
	}

	if err := s.storePendingMFARotation(ctx, pending); err != nil {
		return BeginMFARotationResponse{}, err
	}

	_ = s.lockout.RecordMFASuccess(ctx, pending.StaffAccountID)

	return response, nil
}

func (s *Service) ConfirmSelfMFARotation(
	ctx context.Context,
	accessToken string,
	csrfToken string,
	request ConfirmMFARotationRequest,
	metadata ClientMetadata,
) (
	SelfSecurityMutationResult,
	error,
) {
	accessToken = strings.TrimSpace(accessToken)
	csrfToken = strings.TrimSpace(csrfToken)
	request.RotationToken = strings.TrimSpace(request.RotationToken)
	request.Code = strings.TrimSpace(request.Code)

	if accessToken == "" {
		return SelfSecurityMutationResult{}, ErrInvalidAccessToken
	}
	if csrfToken == "" {
		return SelfSecurityMutationResult{}, ErrInvalidCSRFToken
	}
	if request.RotationToken == "" || request.Code == "" {
		return SelfSecurityMutationResult{}, ErrInvalidRequest
	}

	principal, err := s.AuthenticateAccessTokenWithCSRF(ctx, accessToken, csrfToken)
	if err != nil {
		return SelfSecurityMutationResult{}, err
	}

	blocked, err := s.lockout.IsMFABlocked(ctx, principal.Staff.ID, metadata.IPAddress)
	if err != nil {
		return SelfSecurityMutationResult{}, fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	if blocked {
		return SelfSecurityMutationResult{}, ErrMFABlocked
	}

	pending, err := s.loadPendingMFARotation(ctx, principal.Staff.ID)
	if err != nil {
		return SelfSecurityMutationResult{}, err
	}

	now := s.now().UTC()
	if pending.SessionID != principal.SessionID ||
		!platformsecurity.ConstantTimeEqual(
			pending.RotationTokenHash,
			platformsecurity.HashToken(request.RotationToken),
		) ||
		!pending.ExpiresAt.After(now) {
		return SelfSecurityMutationResult{}, ErrInvalidMFARotation
	}

	secret, err := s.mfa.keyring.DecryptString(
		platformsecurity.EncryptedValue{
			KeyID:      pending.EncryptionKeyID,
			Ciphertext: pending.SecretCiphertext,
		},
		totpAAD(pending.StaffAccountID, pending.CredentialID),
	)
	if err != nil {
		return SelfSecurityMutationResult{}, fmt.Errorf("decrypt pending Admin MFA rotation secret: %w", err)
	}

	cfg := TOTPConfig{
		Algorithm:     pending.Algorithm,
		Digits:        pending.Digits,
		PeriodSeconds: int64(pending.PeriodSeconds),
		Skew:          DefaultTOTPSkew,
	}

	verification, verifyErr := VerifyTOTP(secret, request.Code, now, nil, cfg)
	if verifyErr != nil {
		if errors.Is(verifyErr, ErrInvalidTOTPCode) {
			_ = s.lockout.RecordMFAFailure(ctx, principal.Staff.ID, metadata.IPAddress)
			return SelfSecurityMutationResult{}, ErrInvalidMFACode
		}
		return SelfSecurityMutationResult{}, verifyErr
	}
	if !verification.Valid {
		_ = s.lockout.RecordMFAFailure(ctx, principal.Staff.ID, metadata.IPAddress)
		return SelfSecurityMutationResult{}, ErrInvalidMFACode
	}

	recoveryCodes, recoveryHashes, err := generateRecoveryCodesAndHashes()
	if err != nil {
		return SelfSecurityMutationResult{}, err
	}

	var result SelfSecurityMutationResult
	var publicErr error

	err = platformdatabase.WithinTx(
		ctx,
		s.repository.db,
		func(ctx context.Context, tx pgx.Tx) error {
			session, err := s.repository.LockSelfSecuritySessionByAccessTokenHashTx(
				ctx,
				tx,
				platformsecurity.HashToken(accessToken),
			)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					publicErr = ErrInvalidAccessToken
					return nil
				}
				return err
			}

			if err := validateSelfSecuritySession(session, csrfToken, now); err != nil {
				publicErr = err
				return nil
			}
			if session.ID != pending.SessionID || session.StaffAccountID != pending.StaffAccountID {
				publicErr = ErrInvalidMFARotation
				return nil
			}

			credential, err := s.repository.LockTOTPCredentialTx(ctx, tx, session.StaffAccountID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					publicErr = ErrMFANotEnrolled
					return nil
				}
				return err
			}
			if credential.Status != TOTPCredentialStatusActive || credential.ID != pending.CredentialID {
				publicErr = ErrInvalidMFARotation
				return nil
			}
			if credential.SecretCiphertext == pending.SecretCiphertext && credential.EncryptionKeyID == pending.EncryptionKeyID {
				publicErr = ErrInvalidMFARotation
				return nil
			}

			if err := s.repository.RotateActiveTOTPCredentialTx(
				ctx,
				tx,
				session.StaffAccountID,
				pending,
				verification.Step,
				now,
			); err != nil {
				return err
			}

			if err := s.repository.ReplaceRecoveryCodesTx(ctx, tx, session.StaffAccountID, recoveryHashes); err != nil {
				return err
			}

			if err := s.repository.CancelAllPendingChallengesTx(ctx, tx, session.StaffAccountID); err != nil {
				return err
			}

			if err := s.repository.RevokeOtherStaffSecuritySessionsTx(
				ctx,
				tx,
				session.StaffAccountID,
				session.ID,
				"self_mfa_rotation",
				now,
			); err != nil {
				return err
			}

			material, err := newAdminSessionMaterial(now, &session.RefreshExpiresAt)
			if err != nil {
				return err
			}
			if err := s.repository.RotateAdminSessionTx(
				ctx,
				tx,
				session.ID,
				material,
				metadata.IPAddress,
				metadata.UserAgent,
				now,
			); err != nil {
				return err
			}
			if err := s.repository.UpdateAdminSessionMFAVerifiedAtTx(
				ctx,
				tx,
				session.ID,
				now,
			); err != nil {
				return err
			}

			staffID := session.StaffAccountID
			sessionID := session.ID
			if err := s.repository.InsertSecurityEventTx(
				ctx,
				tx,
				newSecurityEvent(
					SecurityEventMFARotationCompleted,
					SecurityOutcomeSuccess,
					&staffID,
					&sessionID,
					"",
					metadata,
					map[string]any{
						"credential_id":              credential.ID,
						"recovery_codes_regenerated": true,
					},
					now,
				),
			); err != nil {
				return err
			}

			result = SelfSecurityMutationResult{
				Session: SessionResult{
					Principal: AdminPrincipal{
						SessionID:       session.ID,
						AuthenticatedAt: session.AuthenticatedAt,
						MFAVerifiedAt:   now,
						AccessExpiresAt: material.AccessExpiresAt,
					},
					Material: material,
				},
				RecoveryCodes: recoveryCodes,
			}
			return nil
		},
	)
	if err != nil {
		return SelfSecurityMutationResult{}, err
	}
	if publicErr != nil {
		return SelfSecurityMutationResult{}, publicErr
	}

	updatedPrincipal, _, _, err := s.repository.GetPrincipalByAccessTokenHash(
		ctx,
		result.Session.Material.AccessTokenHash,
	)
	if err != nil {
		return SelfSecurityMutationResult{}, err
	}
	result.Session.Principal = updatedPrincipal

	_ = s.deletePendingMFARotation(ctx, pending.StaffAccountID)
	_ = s.lockout.RecordMFASuccess(ctx, pending.StaffAccountID)

	return result, nil
}

func (s *Service) RegenerateSelfRecoveryCodes(
	ctx context.Context,
	accessToken string,
	csrfToken string,
	request RegenerateRecoveryCodesRequest,
	metadata ClientMetadata,
) (
	SelfSecurityMutationResult,
	error,
) {
	accessToken = strings.TrimSpace(accessToken)
	csrfToken = strings.TrimSpace(csrfToken)
	request.Method = strings.ToLower(strings.TrimSpace(request.Method))
	request.Code = strings.TrimSpace(request.Code)

	if accessToken == "" {
		return SelfSecurityMutationResult{}, ErrInvalidAccessToken
	}
	if csrfToken == "" {
		return SelfSecurityMutationResult{}, ErrInvalidCSRFToken
	}
	if request.Password == "" || request.Code == "" || !validSelfSecurityMFAMethod(request.Method) {
		return SelfSecurityMutationResult{}, ErrInvalidRequest
	}

	principal, err := s.AuthenticateAccessTokenWithCSRF(ctx, accessToken, csrfToken)
	if err != nil {
		return SelfSecurityMutationResult{}, err
	}

	blocked, err := s.lockout.IsMFABlocked(ctx, principal.Staff.ID, metadata.IPAddress)
	if err != nil {
		return SelfSecurityMutationResult{}, fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	if blocked {
		return SelfSecurityMutationResult{}, ErrMFABlocked
	}

	recoveryCodes, recoveryHashes, err := generateRecoveryCodesAndHashes()
	if err != nil {
		return SelfSecurityMutationResult{}, err
	}

	now := s.now().UTC()
	var result SelfSecurityMutationResult
	var publicErr error
	var stepUpFailed bool
	staffIDForLockout := principal.Staff.ID

	err = platformdatabase.WithinTx(
		ctx,
		s.repository.db,
		func(ctx context.Context, tx pgx.Tx) error {
			session, err := s.repository.LockSelfSecuritySessionByAccessTokenHashTx(
				ctx,
				tx,
				platformsecurity.HashToken(accessToken),
			)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					publicErr = ErrInvalidAccessToken
					return nil
				}
				return err
			}
			if err := validateSelfSecuritySession(session, csrfToken, now); err != nil {
				publicErr = err
				return nil
			}
			if session.ID != principal.SessionID || session.StaffAccountID != principal.Staff.ID {
				publicErr = ErrInvalidAccessToken
				return nil
			}

			validPassword, replacementHash, err := verifyStoredPassword(request.Password, session.PasswordHash)
			if err != nil {
				return err
			}
			if !validPassword {
				stepUpFailed = true
				publicErr = ErrInvalidCredentials
				return nil
			}

			credential, err := s.repository.LockTOTPCredentialTx(ctx, tx, session.StaffAccountID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					publicErr = ErrMFANotEnrolled
					return nil
				}
				return err
			}
			if credential.Status != TOTPCredentialStatusActive {
				publicErr = ErrMFANotEnrolled
				return nil
			}

			usedRecoveryCode, valid, err := s.verifySelfSecurityStepUpTx(
				ctx,
				tx,
				session,
				credential,
				request.Method,
				request.Code,
				metadata,
				now,
			)
			if err != nil {
				return err
			}
			if !valid {
				stepUpFailed = true
				publicErr = ErrInvalidMFACode
				return nil
			}

			if replacementHash != "" {
				if err := s.repository.UpdatePasswordHashTx(ctx, tx, session.StaffAccountID, replacementHash); err != nil {
					return err
				}
			}

			if err := s.repository.ReplaceRecoveryCodesTx(ctx, tx, session.StaffAccountID, recoveryHashes); err != nil {
				return err
			}
			if err := s.repository.CancelAllPendingChallengesTx(ctx, tx, session.StaffAccountID); err != nil {
				return err
			}
			if err := s.repository.RevokeOtherStaffSecuritySessionsTx(
				ctx,
				tx,
				session.StaffAccountID,
				session.ID,
				"recovery_codes_regenerated",
				now,
			); err != nil {
				return err
			}

			material, err := newAdminSessionMaterial(now, &session.RefreshExpiresAt)
			if err != nil {
				return err
			}
			if err := s.repository.RotateAdminSessionTx(
				ctx,
				tx,
				session.ID,
				material,
				metadata.IPAddress,
				metadata.UserAgent,
				now,
			); err != nil {
				return err
			}
			if err := s.repository.UpdateAdminSessionMFAVerifiedAtTx(
				ctx,
				tx,
				session.ID,
				now,
			); err != nil {
				return err
			}

			staffID := session.StaffAccountID
			sessionID := session.ID
			if err := s.repository.InsertSecurityEventTx(
				ctx,
				tx,
				newSecurityEvent(
					SecurityEventRecoveryCodesRegenerated,
					SecurityOutcomeSuccess,
					&staffID,
					&sessionID,
					"",
					metadata,
					map[string]any{
						"method":             request.Method,
						"used_recovery_code": usedRecoveryCode,
					},
					now,
				),
			); err != nil {
				return err
			}

			result = SelfSecurityMutationResult{
				Session: SessionResult{
					Principal: AdminPrincipal{
						SessionID:       session.ID,
						AuthenticatedAt: session.AuthenticatedAt,
						MFAVerifiedAt:   now,
						AccessExpiresAt: material.AccessExpiresAt,
					},
					Material: material,
				},
				RecoveryCodes: recoveryCodes,
			}
			return nil
		},
	)
	if err != nil {
		return SelfSecurityMutationResult{}, err
	}

	if stepUpFailed && staffIDForLockout != "" {
		_ = s.lockout.RecordMFAFailure(ctx, staffIDForLockout, metadata.IPAddress)
	}
	if publicErr != nil {
		return SelfSecurityMutationResult{}, publicErr
	}

	_ = s.lockout.RecordMFASuccess(ctx, staffIDForLockout)

	updatedPrincipal, _, _, err := s.repository.GetPrincipalByAccessTokenHash(
		ctx,
		result.Session.Material.AccessTokenHash,
	)
	if err != nil {
		return SelfSecurityMutationResult{}, err
	}
	result.Session.Principal = updatedPrincipal

	return result, nil
}

func (s *Service) verifySelfSecurityStepUpTx(
	ctx context.Context,
	tx pgx.Tx,
	session SelfSecuritySessionRecord,
	credential TOTPCredential,
	method string,
	code string,
	metadata ClientMetadata,
	now time.Time,
) (
	bool,
	bool,
	error,
) {
	switch method {
	case MFAMethodTOTP:
		secret, err := s.mfa.decryptCredential(credential)
		if err != nil {
			return false, false, err
		}

		verification, verifyErr := VerifyTOTP(
			secret,
			code,
			now,
			credential.LastAcceptedStep,
			totpConfigFromCredential(credential),
		)
		if verifyErr != nil {
			if errors.Is(verifyErr, ErrInvalidTOTPCode) {
				return false, false, nil
			}
			return false, false, verifyErr
		}
		if !verification.Valid {
			return false, false, nil
		}

		if err := s.repository.UpdateAcceptedTOTPStepTx(
			ctx,
			tx,
			session.StaffAccountID,
			verification.Step,
			now,
		); err != nil {
			if errors.Is(err, ErrInvalidMFACode) {
				return false, false, nil
			}
			return false, false, err
		}

		return false, true, nil

	case MFAMethodRecoveryCode:
		hash, err := HashRecoveryCode(code)
		if err != nil {
			return false, false, nil
		}
		consumed, err := s.repository.ConsumeRecoveryCodeTx(
			ctx,
			tx,
			session.StaffAccountID,
			hash,
			now,
		)
		if err != nil {
			return false, false, err
		}
		if !consumed {
			return false, false, nil
		}

		staffID := session.StaffAccountID
		sessionID := session.ID
		if err := s.repository.InsertSecurityEventTx(
			ctx,
			tx,
			newSecurityEvent(
				SecurityEventRecoveryCodeUsed,
				SecurityOutcomeSuccess,
				&staffID,
				&sessionID,
				"",
				metadata,
				map[string]any{"stage": "self_security_step_up"},
				now,
			),
		); err != nil {
			return false, false, err
		}

		return true, true, nil
	}

	return false, false, nil
}

func validSelfSecurityMFAMethod(method string) bool {
	return method == MFAMethodTOTP || method == MFAMethodRecoveryCode
}

func validateSelfSecuritySession(
	session SelfSecuritySessionRecord,
	csrfToken string,
	now time.Time,
) error {
	if session.RevokedAt != nil {
		return ErrSessionRevoked
	}
	if session.StaffStatus != "active" {
		return ErrAdminAccountDisabled
	}
	if !session.HasPanelAccess {
		return ErrAdminPanelAccessRequired
	}
	if !session.AccessExpiresAt.After(now) {
		return ErrInvalidAccessToken
	}
	if !platformsecurity.VerifyCSRFToken(csrfToken, session.CSRFTokenHash) {
		return ErrInvalidCSRFToken
	}
	return nil
}

func (s *Service) storePendingMFARotation(
	ctx context.Context,
	pending pendingMFARotation,
) error {
	payload, err := json.Marshal(pending)
	if err != nil {
		return fmt.Errorf("encode pending Admin MFA rotation: %w", err)
	}

	if err := s.lockout.redis.Set(
		ctx,
		selfMFARotationRedisKey(pending.StaffAccountID),
		payload,
		AdminMFARotationTTL,
	).Err(); err != nil {
		return fmt.Errorf("%w: store pending Admin MFA rotation: %v", ErrAuthUnavailable, err)
	}

	return nil
}

func (s *Service) loadPendingMFARotation(
	ctx context.Context,
	staffAccountID string,
) (
	pendingMFARotation,
	error,
) {
	payload, err := s.lockout.redis.Get(
		ctx,
		selfMFARotationRedisKey(staffAccountID),
	).Bytes()
	if errors.Is(err, redis.Nil) {
		return pendingMFARotation{}, ErrInvalidMFARotation
	}
	if err != nil {
		return pendingMFARotation{}, fmt.Errorf(
			"%w: load pending Admin MFA rotation: %v",
			ErrAuthUnavailable,
			err,
		)
	}

	var pending pendingMFARotation
	if err := json.Unmarshal(payload, &pending); err != nil {
		return pendingMFARotation{}, fmt.Errorf("decode pending Admin MFA rotation: %w", err)
	}

	return pending, nil
}

func (s *Service) deletePendingMFARotation(
	ctx context.Context,
	staffAccountID string,
) error {
	if err := s.lockout.redis.Del(
		ctx,
		selfMFARotationRedisKey(staffAccountID),
	).Err(); err != nil {
		return fmt.Errorf("delete pending Admin MFA rotation: %w", err)
	}
	return nil
}

func selfMFARotationRedisKey(staffAccountID string) string {
	return "adminauth:self-mfa-rotation:" + strings.TrimSpace(staffAccountID)
}
