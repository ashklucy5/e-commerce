package auth

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalidCustomerSessionID = errors.New(
		"invalid customer session id",
	)

	ErrCustomerSessionNotFound = errors.New(
		"customer session not found",
	)

	customerSessionUUIDPattern = regexp.MustCompile(
		`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
	)
)

type CustomerSession struct {
	ID string `json:"id"`

	Current bool `json:"current"`

	CreatedAt time.Time `json:"created_at"`

	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	AccessExpiresAt time.Time `json:"access_expires_at"`

	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type CustomerSecurityOverview struct {
	CustomerID string `json:"customer_id"`

	CurrentSessionID string `json:"current_session_id"`

	ActiveSessionCount int64 `json:"active_session_count"`

	PasswordChangeAvailable bool `json:"password_change_available"`

	PasswordRecoveryAvailable bool `json:"password_recovery_available"`
}

type RevokeOtherSessionsResult struct {
	RevokedSessions int64 `json:"revoked_sessions"`
}

func (s *Service) GetCustomerSecurityOverview(
	ctx context.Context,
	customerID string,
	accessToken string,
) (CustomerSecurityOverview, error) {
	accessTokenHash, err :=
		normalizeCustomerSessionAccessToken(
			accessToken,
		)
	if err != nil {
		return CustomerSecurityOverview{},
			err
	}

	currentSessionID, activeSessionCount, err :=
		s.repository.GetCustomerSecuritySnapshot(
			ctx,
			customerID,
			accessTokenHash,
		)
	if err != nil {
		return CustomerSecurityOverview{},
			err
	}

	return CustomerSecurityOverview{
		CustomerID: customerID,

		CurrentSessionID: currentSessionID,

		ActiveSessionCount: activeSessionCount,

		PasswordChangeAvailable: true,

		PasswordRecoveryAvailable: s.otpPolicy.Enabled &&
			s.passwordResetOTP != nil,
	}, nil
}

func (s *Service) ListCustomerSessions(
	ctx context.Context,
	customerID string,
	accessToken string,
) ([]CustomerSession, error) {
	accessTokenHash, err :=
		normalizeCustomerSessionAccessToken(
			accessToken,
		)
	if err != nil {
		return nil,
			err
	}

	return s.repository.ListCustomerSessions(
		ctx,
		customerID,
		accessTokenHash,
	)
}

func (s *Service) RevokeCustomerSession(
	ctx context.Context,
	customerID string,
	sessionID string,
) error {
	sessionID =
		strings.ToLower(
			strings.TrimSpace(
				sessionID,
			),
		)

	if !customerSessionUUIDPattern.MatchString(
		sessionID,
	) {
		return ErrInvalidCustomerSessionID
	}

	return s.repository.RevokeCustomerSession(
		ctx,
		customerID,
		sessionID,
	)
}

func (s *Service) RevokeOtherCustomerSessions(
	ctx context.Context,
	customerID string,
	accessToken string,
) (RevokeOtherSessionsResult, error) {
	accessTokenHash, err :=
		normalizeCustomerSessionAccessToken(
			accessToken,
		)
	if err != nil {
		return RevokeOtherSessionsResult{},
			err
	}

	revoked, err :=
		s.repository.RevokeOtherActiveCustomerSessions(
			ctx,
			customerID,
			accessTokenHash,
		)
	if err != nil {
		return RevokeOtherSessionsResult{},
			err
	}

	return RevokeOtherSessionsResult{
		RevokedSessions: revoked,
	}, nil
}

func normalizeCustomerSessionAccessToken(
	accessToken string,
) (string, error) {
	accessToken =
		strings.TrimSpace(
			accessToken,
		)

	if accessToken == "" {
		return "",
			ErrInvalidAccessToken
	}

	return hashToken(
		accessToken,
	), nil
}
