package adminauth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	platformdatabase "project.local/commerce-api/internal/platform/database"
)

var (
	ErrAdminSessionNotFound = errors.New(
		"Admin session was not found",
	)

	ErrCannotRevokeCurrentAdminSession = errors.New(
		"current Admin session must be ended with logout",
	)
)

type SelfSecurityMFAOverview struct {
	Enabled bool `json:"enabled"`

	Label string `json:"label"`

	VerifiedAt *time.Time `json:"verified_at"`

	LastUsedAt *time.Time `json:"last_used_at"`

	RecoveryCodesRemaining int64 `json:"recovery_codes_remaining"`
}

type SelfSecuritySessionOverview struct {
	ID string `json:"id"`

	Current bool `json:"current"`

	AuthenticatedAt time.Time `json:"authenticated_at"`

	MFAVerifiedAt time.Time `json:"mfa_verified_at"`

	LastUsedAt *time.Time `json:"last_used_at"`

	LastRotatedAt *time.Time `json:"last_rotated_at"`

	CreatedIP string `json:"created_ip"`

	LastIP string `json:"last_ip"`

	UserAgent string `json:"user_agent"`

	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type SelfSecurityOverviewResponse struct {
	MFA SelfSecurityMFAOverview `json:"mfa"`

	Sessions []SelfSecuritySessionOverview `json:"sessions"`
}

type RevokeOtherSelfSecuritySessionsResponse struct {
	RevokedSessions int64 `json:"revoked_sessions"`
}

func (s *Service) SelfSecurityOverview(
	ctx context.Context,
	accessToken string,
) (
	SelfSecurityOverviewResponse,
	error,
) {
	principal, err :=
		s.AuthenticateAccessToken(
			ctx,
			accessToken,
		)
	if err != nil {
		return SelfSecurityOverviewResponse{},
			err
	}

	mfa,
		sessions,
		err :=
		s.repository.GetSelfSecurityOverview(
			ctx,
			principal.Staff.ID,
			principal.SessionID,
			s.now().UTC(),
		)
	if err != nil {
		return SelfSecurityOverviewResponse{},
			err
	}

	return SelfSecurityOverviewResponse{
		MFA: mfa,

		Sessions: sessions,
	}, nil
}

func (s *Service) RevokeSelfSecuritySession(
	ctx context.Context,
	accessToken string,
	csrfToken string,
	targetSessionID string,
	metadata ClientMetadata,
) error {
	targetSessionID =
		strings.TrimSpace(
			targetSessionID,
		)

	if targetSessionID == "" {
		return ErrInvalidRequest
	}

	principal, err :=
		s.AuthenticateAccessTokenWithCSRF(
			ctx,
			accessToken,
			csrfToken,
		)
	if err != nil {
		return err
	}

	if targetSessionID == principal.SessionID {
		return ErrCannotRevokeCurrentAdminSession
	}

	now := s.now().UTC()

	var publicErr error

	err = platformdatabase.WithinTx(
		ctx,
		s.repository.db,
		func(
			ctx context.Context,
			tx pgx.Tx,
		) error {
			revokedSessionID, err :=
				s.repository.RevokeOwnedAdminSessionTx(
					ctx,
					tx,
					principal.Staff.ID,
					targetSessionID,
					principal.SessionID,
					now,
				)
			if err != nil {
				if errors.Is(
					err,
					pgx.ErrNoRows,
				) {
					publicErr =
						ErrAdminSessionNotFound

					return nil
				}

				return err
			}

			staffID := principal.Staff.ID

			if err :=
				s.repository.InsertSecurityEventTx(
					ctx,
					tx,
					newSecurityEvent(
						SecurityEventSessionRevoked,
						SecurityOutcomeSuccess,
						&staffID,
						&revokedSessionID,
						"",
						metadata,
						map[string]any{
							"reason": "self_session_revoke",

							"actor_session_id": principal.SessionID,
						},
						now,
					),
				); err != nil {

				return err
			}

			return nil
		},
	)
	if err != nil {
		return err
	}

	if publicErr != nil {
		return publicErr
	}

	return nil
}

func (s *Service) RevokeOtherSelfSecuritySessions(
	ctx context.Context,
	accessToken string,
	csrfToken string,
	metadata ClientMetadata,
) (
	RevokeOtherSelfSecuritySessionsResponse,
	error,
) {
	principal, err :=
		s.AuthenticateAccessTokenWithCSRF(
			ctx,
			accessToken,
			csrfToken,
		)
	if err != nil {
		return RevokeOtherSelfSecuritySessionsResponse{},
			err
	}

	now := s.now().UTC()

	var revoked int64

	err = platformdatabase.WithinTx(
		ctx,
		s.repository.db,
		func(
			ctx context.Context,
			tx pgx.Tx,
		) error {
			count, err :=
				s.repository.RevokeOtherOwnedAdminSessionsTx(
					ctx,
					tx,
					principal.Staff.ID,
					principal.SessionID,
					now,
				)
			if err != nil {
				return err
			}

			revoked = count

			staffID := principal.Staff.ID
			sessionID := principal.SessionID

			if err :=
				s.repository.InsertSecurityEventTx(
					ctx,
					tx,
					newSecurityEvent(
						SecurityEventSessionRevoked,
						SecurityOutcomeSuccess,
						&staffID,
						&sessionID,
						"",
						metadata,
						map[string]any{
							"reason": "self_revoke_other_sessions",

							"revoked_sessions": revoked,
						},
						now,
					),
				); err != nil {

				return err
			}

			return nil
		},
	)
	if err != nil {
		return RevokeOtherSelfSecuritySessionsResponse{},
			err
	}

	return RevokeOtherSelfSecuritySessionsResponse{
		RevokedSessions: revoked,
	}, nil
}
