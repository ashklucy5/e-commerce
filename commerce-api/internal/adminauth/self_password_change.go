package adminauth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const SecurityEventSelfPasswordChanged = "self_password_changed"

var ErrPasswordUnchanged = errors.New(
	"new Admin password must be different from the current password",
)

type ChangeSelfPasswordRequest struct {
	SecurityStepUpRequest

	NewPassword string `json:"new_password"`
}

type ChangeSelfPasswordResponse struct {
	Principal AdminPrincipal `json:"principal"`

	CSRFToken string `json:"csrf_token"`
}

// ChangeSelfPassword changes the password for the currently authenticated
// Admin-panel principal. It requires the current password plus a fresh TOTP
// code or one unused recovery code. The current Admin session is rotated and
// every other staff/Admin session is revoked.
func (s *Service) ChangeSelfPassword(
	ctx context.Context,
	accessToken string,
	csrfToken string,
	request ChangeSelfPasswordRequest,
	metadata ClientMetadata,
) (
	SessionResult,
	error,
) {
	accessToken = strings.TrimSpace(accessToken)
	csrfToken = strings.TrimSpace(csrfToken)
	request.Method = strings.ToLower(strings.TrimSpace(request.Method))
	request.Code = strings.TrimSpace(request.Code)

	if accessToken == "" {
		return SessionResult{}, ErrInvalidAccessToken
	}
	if csrfToken == "" {
		return SessionResult{}, ErrInvalidCSRFToken
	}
	if request.Password == "" || request.Code == "" || !validSelfSecurityMFAMethod(request.Method) {
		return SessionResult{}, ErrInvalidRequest
	}
	if err := ValidateNewPassword(request.NewPassword); err != nil {
		return SessionResult{}, err
	}

	principal, err := s.AuthenticateAccessTokenWithCSRF(ctx, accessToken, csrfToken)
	if err != nil {
		return SessionResult{}, err
	}

	blocked, err := s.lockout.IsMFABlocked(ctx, principal.Staff.ID, metadata.IPAddress)
	if err != nil {
		return SessionResult{}, fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	if blocked {
		return SessionResult{}, ErrMFABlocked
	}

	newPasswordHash, err := HashNewPassword(request.NewPassword)
	if err != nil {
		return SessionResult{}, err
	}

	now := s.now().UTC()
	var result SessionResult
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

			validPassword, _, err := verifyStoredPassword(request.Password, session.PasswordHash)
			if err != nil {
				return err
			}
			if !validPassword {
				stepUpFailed = true
				publicErr = ErrInvalidCredentials
				return nil
			}

			unchanged, _, err := verifyStoredPassword(request.NewPassword, session.PasswordHash)
			if err != nil {
				return err
			}
			if unchanged {
				publicErr = ErrPasswordUnchanged
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

			if err := s.repository.UpdatePasswordHashTx(
				ctx,
				tx,
				session.StaffAccountID,
				newPasswordHash,
			); err != nil {
				return err
			}

			if err := s.repository.CancelAllPendingChallengesTx(
				ctx,
				tx,
				session.StaffAccountID,
			); err != nil {
				return err
			}

			if err := s.repository.RevokeOtherStaffSecuritySessionsTx(
				ctx,
				tx,
				session.StaffAccountID,
				session.ID,
				"self_password_changed",
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
					SecurityEventSelfPasswordChanged,
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

			result = SessionResult{
				Principal: AdminPrincipal{
					SessionID:       session.ID,
					AuthenticatedAt: session.AuthenticatedAt,
					MFAVerifiedAt:   now,
					AccessExpiresAt: material.AccessExpiresAt,
				},
				Material: material,
			}

			return nil
		},
	)
	if err != nil {
		return SessionResult{}, err
	}

	if stepUpFailed && staffIDForLockout != "" {
		_ = s.lockout.RecordMFAFailure(ctx, staffIDForLockout, metadata.IPAddress)
	}
	if publicErr != nil {
		return SessionResult{}, publicErr
	}

	_ = s.lockout.RecordMFASuccess(ctx, staffIDForLockout)

	updatedPrincipal, _, _, err := s.repository.GetPrincipalByAccessTokenHash(
		ctx,
		result.Material.AccessTokenHash,
	)
	if err != nil {
		return SessionResult{}, err
	}
	result.Principal = updatedPrincipal

	return result, nil
}
