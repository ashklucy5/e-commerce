package auth

import "time"

type Customer struct {
	ID           string    `json:"id"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email,omitempty"`
	FullName     string    `json:"full_name"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	PasswordHash string    `json:"-"`
}

type Tokens struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	TokenType        string    `json:"token_type"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type AuthResult struct {
	Customer Customer `json:"customer"`
	Tokens   Tokens   `json:"tokens"`
}

type RegisterResult struct {
	VerificationRequired bool `json:"verification_required"`

	Verification *RegistrationChallenge `json:"verification,omitempty"`

	Customer *Customer `json:"customer,omitempty"`

	Tokens *Tokens `json:"tokens,omitempty"`
}

type RegistrationChallenge struct {
	VerificationID    string    `json:"verification_id"`
	Phone             string    `json:"phone"`
	ExpiresAt         time.Time `json:"expires_at"`
	ResendAvailableAt time.Time `json:"resend_available_at"`
}

type pendingRegistration struct {
	VerificationID string `json:"verification_id"`

	Phone string `json:"phone"`
	Email string `json:"email"`

	FullName string `json:"full_name"`

	PasswordHash string `json:"password_hash"`

	OTPHash string `json:"otp_hash"`

	ExpiresAt         time.Time `json:"expires_at"`
	ResendAvailableAt time.Time `json:"resend_available_at"`

	Attempts    int `json:"attempts"`
	ResendCount int `json:"resend_count"`
}

type PasswordResetChallenge struct {
	VerificationID string `json:"verification_id"`

	ExpiresAt time.Time `json:"expires_at"`

	ResendAvailableAt time.Time `json:"resend_available_at"`
}

type PasswordResetGrant struct {
	ResetToken string `json:"reset_token"`

	ExpiresAt time.Time `json:"expires_at"`
}

type pendingPasswordReset struct {
	VerificationID string `json:"verification_id"`

	Phone string `json:"phone"`

	CustomerID string `json:"customer_id"`

	OTPHash string `json:"otp_hash"`

	ExpiresAt time.Time `json:"expires_at"`

	ResendAvailableAt time.Time `json:"resend_available_at"`

	Attempts int `json:"attempts"`

	ResendCount int `json:"resend_count"`
}

type passwordResetGrantRecord struct {
	CustomerID string `json:"customer_id"`

	ExpiresAt time.Time `json:"expires_at"`
}

type sessionRecord struct {
	ID               string
	CustomerID       string
	RefreshExpiresAt time.Time
	Customer         Customer
}
