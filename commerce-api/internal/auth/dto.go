package auth

type RegisterRequest struct {
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type RegisterVerifyRequest struct {
	VerificationID string `json:"verification_id"`
	Code           string `json:"code"`
}

type RegisterResendRequest struct {
	VerificationID string `json:"verification_id"`
}

type LoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Phone string `json:"phone"`
}

type ForgotPasswordVerifyRequest struct {
	VerificationID string `json:"verification_id"`
	Code           string `json:"code"`
}

type ForgotPasswordResendRequest struct {
	VerificationID string `json:"verification_id"`
}

type ResetPasswordRequest struct {
	ResetToken string `json:"reset_token"`

	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`

	/*
		nil means the secure default: revoke other devices.
	*/
	RevokeOtherSessions *bool `json:"revoke_other_sessions"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`

	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`

	RevokeOtherSessions *bool `json:"revoke_other_sessions"`
}
