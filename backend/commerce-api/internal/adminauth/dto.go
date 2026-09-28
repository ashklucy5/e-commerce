package adminauth

import "time"

const (
	MFAMethodTOTP = "totp"

	MFAMethodRecoveryCode = "recovery_code"
)

type LoginRequest struct {
	Identifier string `json:"identifier"`

	Password string `json:"password"`
}

type LoginChallengeResponse struct {
	ChallengeToken string `json:"challenge_token"`

	ChallengeExpiresAt time.Time `json:"challenge_expires_at"`

	MFAEnrollmentRequired bool `json:"mfa_enrollment_required"`
}

type MFAEnrollRequest struct {
	ChallengeToken string `json:"challenge_token"`

	Label string `json:"label"`
}

type MFAEnrollResponse struct {
	Enrollment MFAEnrollment `json:"enrollment"`
}

type MFAConfirmEnrollmentRequest struct {
	ChallengeToken string `json:"challenge_token"`

	Code string `json:"code"`
}

type MFAVerifyRequest struct {
	ChallengeToken string `json:"challenge_token"`

	Method string `json:"method"`

	Code string `json:"code"`
}

type SessionResponse struct {
	Principal AdminPrincipal `json:"principal"`

	CSRFToken string `json:"csrf_token"`
}

type MFAEnrollmentCompleteResponse struct {
	Principal AdminPrincipal `json:"principal"`

	CSRFToken string `json:"csrf_token"`

	RecoveryCodes []string `json:"recovery_codes"`
}

type RefreshResponse struct {
	Principal AdminPrincipal `json:"principal"`

	CSRFToken string `json:"csrf_token"`
}

type MeResponse struct {
	Principal AdminPrincipal `json:"principal"`
}
