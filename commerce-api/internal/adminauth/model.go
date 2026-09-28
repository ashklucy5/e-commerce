package adminauth

import "time"

const (
	TOTPCredentialStatusPending = "pending"

	TOTPCredentialStatusActive = "active"

	TOTPCredentialStatusDisabled = "disabled"
)

type MFAAccount struct {
	ID string

	StaffCode string

	Email string

	Status string

	HasPanelAccess bool
}

type TOTPCredential struct {
	ID string

	StaffAccountID string

	Label string

	SecretCiphertext string

	EncryptionKeyID string

	Algorithm string

	Digits int

	PeriodSeconds int

	Status string

	LastAcceptedStep *int64

	VerifiedAt *time.Time

	LastUsedAt *time.Time

	DisabledAt *time.Time

	CreatedAt time.Time

	UpdatedAt time.Time
}

type MFAEnrollment struct {
	CredentialID string `json:"credential_id"`

	Label string `json:"label"`

	Secret string `json:"secret"`

	EnrollmentURI string `json:"enrollment_uri"`

	Algorithm string `json:"algorithm"`

	Digits int `json:"digits"`

	PeriodSeconds int `json:"period_seconds"`
}

type MFAEnrollmentConfirmation struct {
	CredentialID string `json:"credential_id"`

	VerifiedAt time.Time `json:"verified_at"`

	// Recovery codes are returned exactly once after successful MFA
	// enrollment. Only their hashes are persisted.
	RecoveryCodes []string `json:"recovery_codes"`
}
