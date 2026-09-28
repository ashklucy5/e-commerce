package adminauth

import (
	"time"

	"project.local/commerce-api/internal/staff"
)

const (
	ChallengePurposeLogin  = "login"
	ChallengePurposeReauth = "reauth"

	ChallengeStatusPending   = "pending"
	ChallengeStatusVerified  = "verified"
	ChallengeStatusConsumed  = "consumed"
	ChallengeStatusExpired   = "expired"
	ChallengeStatusLocked    = "locked"
	ChallengeStatusCancelled = "cancelled"
)

type LoginAccount struct {
	staff.Account

	PasswordHash string

	HasPanelAccess bool

	HasActiveMFA bool
}

type LoginChallenge struct {
	ID string

	StaffAccountID string

	ChallengeTokenHash string

	Purpose string

	Status string

	FailedAttempts int

	MaxAttempts int

	ExpiresAt time.Time

	VerifiedAt *time.Time

	ConsumedAt *time.Time

	ClientIP string

	UserAgent string

	RequestID string

	CreatedAt time.Time

	UpdatedAt time.Time
}

type AdminSessionRecord struct {
	ID string

	StaffAccountID string

	LoginChallengeID string

	AccessTokenHash string

	RefreshTokenHash string

	CSRFTokenHash string

	RefreshGeneration int

	AccessExpiresAt time.Time

	RefreshExpiresAt time.Time

	AuthenticatedAt time.Time

	MFAVerifiedAt time.Time

	LastUsedAt *time.Time

	LastRotatedAt *time.Time

	CreatedIP string

	LastIP string

	UserAgent string

	RevokedAt *time.Time

	RevokeReason string

	StaffStatus string
}

type AdminSessionMaterial struct {
	AccessToken string

	RefreshToken string

	CSRFToken string

	AccessTokenHash string

	RefreshTokenHash string

	CSRFTokenHash string

	AccessExpiresAt time.Time

	RefreshExpiresAt time.Time
}

type AdminPrincipal struct {
	SessionID string `json:"session_id"`

	Staff staff.Account `json:"staff"`

	AuthenticatedAt time.Time `json:"authenticated_at"`

	MFAVerifiedAt time.Time `json:"mfa_verified_at"`

	AccessExpiresAt time.Time `json:"access_expires_at"`
}

type SecurityEvent struct {
	StaffAccountID *string

	AdminSessionID *string

	EventType string

	Outcome string

	IdentifierHash *string

	RequestID string

	IPAddress string

	UserAgent string

	Details map[string]any

	OccurredAt time.Time
}

type ClientMetadata struct {
	IPAddress string

	UserAgent string

	RequestID string
}

type SessionResult struct {
	Principal AdminPrincipal

	Material AdminSessionMaterial
}

type EnrollmentSessionResult struct {
	Session SessionResult

	RecoveryCodes []string
}
