package adminauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	platformdatabase "project.local/commerce-api/internal/platform/database"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

const (
	AdminRecoverySessionTTL = 10 * time.Minute

	SecurityEventRecoveryPasswordReset = "recovery_password_reset"
)

var (
	ErrRecoverySessionRequired = errors.New(
		"Admin recovery-code session is required",
	)

	ErrRecoverySessionExpired = errors.New(
		"Admin recovery-code session has expired",
	)
)

type RecoveryPasswordResetRequest struct {
	NewPassword string `json:"new_password"`
}

type RecoveryPasswordResetResponse struct {
	ChallengeToken string `json:"challenge_token"`

	ChallengeExpiresAt time.Time `json:"challenge_expires_at"`

	MFAEnrollmentRequired bool `json:"mfa_enrollment_required"`
}

// ResetPasswordFromRecoverySession allows a staff member who authenticated
// with password + a one-time MFA recovery code to replace their password
// without re-entering the old password or presenting another MFA code.
//
// The recovery code remains single-use. Authorization is derived from the
// exact Admin session that consumed it and is accepted for only a short
// window. On success, the password changes, old MFA/recovery material is
// invalidated, all existing sessions are revoked, and a one-time challenge
// is returned for fresh MFA enrollment.
func (s *Service) ResetPasswordFromRecoverySession(
	ctx context.Context,
	accessToken string,
	csrfToken string,
	request RecoveryPasswordResetRequest,
	metadata ClientMetadata,
) (
	RecoveryPasswordResetResponse,
	error,
) {
	accessToken =
		strings.TrimSpace(
			accessToken,
		)

	csrfToken =
		strings.TrimSpace(
			csrfToken,
		)

	if accessToken == "" {
		return RecoveryPasswordResetResponse{},
			ErrInvalidAccessToken
	}

	if csrfToken == "" {
		return RecoveryPasswordResetResponse{},
			ErrInvalidCSRFToken
	}

	if err :=
		ValidateNewPassword(
			request.NewPassword,
		); err != nil {

		return RecoveryPasswordResetResponse{},
			err
	}

	// Authenticate once before doing the expensive password hash. The same
	// session is locked and revalidated again inside the transaction below.
	if _, err :=
		s.AuthenticateAccessTokenWithCSRF(
			ctx,
			accessToken,
			csrfToken,
		); err != nil {

		return RecoveryPasswordResetResponse{},
			err
	}

	passwordHash, err :=
		HashNewPassword(
			request.NewPassword,
		)
	if err != nil {
		return RecoveryPasswordResetResponse{},
			err
	}

	now :=
		s.now().UTC()

	challengeToken, err :=
		platformsecurity.RandomURLSafe(
			32,
		)
	if err != nil {
		return RecoveryPasswordResetResponse{},
			fmt.Errorf(
				"generate Admin recovery MFA challenge: %w",
				err,
			)
	}

	challengeExpiresAt :=
		now.Add(
			AdminLoginChallengeTTL,
		)

	var response RecoveryPasswordResetResponse

	var publicErr error

	err =
		platformdatabase.WithinTx(
			ctx,
			s.repository.db,
			func(
				ctx context.Context,
				tx pgx.Tx,
			) error {
				session, err :=
					s.repository.
						LockRecoverySessionByAccessTokenHashTx(
							ctx,
							tx,
							platformsecurity.HashToken(
								accessToken,
							),
						)
				if err != nil {
					if errors.Is(
						err,
						pgx.ErrNoRows,
					) {
						publicErr =
							ErrInvalidAccessToken

						return nil
					}

					return err
				}

				if session.RevokedAt != nil {
					publicErr =
						ErrSessionRevoked

					return nil
				}

				if session.StaffStatus != "active" {
					publicErr =
						ErrAdminAccountDisabled

					return nil
				}

				if !session.HasPanelAccess {
					publicErr =
						ErrAdminPanelAccessRequired

					return nil
				}

				if !session.AccessExpiresAt.After(
					now,
				) {
					publicErr =
						ErrInvalidAccessToken

					return nil
				}

				if !platformsecurity.VerifyCSRFToken(
					csrfToken,
					session.CSRFTokenHash,
				) {
					publicErr =
						ErrInvalidCSRFToken

					return nil
				}

				if session.RecoveryVerifiedAt == nil {
					publicErr =
						ErrRecoverySessionRequired

					return nil
				}

				recoveryVerifiedAt :=
					session.RecoveryVerifiedAt.UTC()

				if recoveryVerifiedAt.After(
					now,
				) ||
					now.Sub(
						recoveryVerifiedAt,
					) > AdminRecoverySessionTTL {

					publicErr =
						ErrRecoverySessionExpired

					return nil
				}

				if err :=
					s.repository.
						UpdatePasswordHashTx(
							ctx,
							tx,
							session.StaffAccountID,
							passwordHash,
						); err != nil {

					return err
				}

				if err :=
					s.repository.
						DisableMFAForRecoveryTx(
							ctx,
							tx,
							session.StaffAccountID,
							now,
						); err != nil {

					return err
				}

				if err :=
					s.repository.
						CancelAllPendingChallengesTx(
							ctx,
							tx,
							session.StaffAccountID,
						); err != nil {

					return err
				}

				challenge, err :=
					s.repository.
						CreateLoginChallengeTx(
							ctx,
							tx,
							session.StaffAccountID,
							platformsecurity.HashToken(
								challengeToken,
							),
							ChallengePurposeReauth,
							challengeExpiresAt,
							AdminChallengeMaxAttempts,
							metadata,
						)
				if err != nil {
					return err
				}

				if err :=
					s.repository.
						RevokeAllStaffSecuritySessionsTx(
							ctx,
							tx,
							session.StaffAccountID,
							"recovery_password_reset",
							now,
						); err != nil {

					return err
				}

				staffID :=
					session.StaffAccountID

				sessionID :=
					session.ID

				if err :=
					s.repository.
						InsertSecurityEventTx(
							ctx,
							tx,
							newSecurityEvent(
								SecurityEventRecoveryPasswordReset,
								SecurityOutcomeSuccess,
								&staffID,
								&sessionID,
								"",
								metadata,
								map[string]any{
									"challenge_id": challenge.ID,

									"recovery_verified_at": recoveryVerifiedAt,

									"mfa_reenrollment_required": true,
								},
								now,
							),
						); err != nil {

					return err
				}

				response =
					RecoveryPasswordResetResponse{
						ChallengeToken: challengeToken,

						ChallengeExpiresAt: challenge.ExpiresAt,

						MFAEnrollmentRequired: true,
					}

				return nil
			},
		)
	if err != nil {
		return RecoveryPasswordResetResponse{},
			err
	}

	if publicErr != nil {
		return RecoveryPasswordResetResponse{},
			publicErr
	}

	return response,
		nil
}
