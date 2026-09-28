package staff

import "time"

type Account struct {
	ID        string `json:"id"`
	StaffCode string `json:"staff_code"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone,omitempty"`
	Status    string `json:"status"`

	Roles []string `json:"roles"`

	Permissions []string `json:"permissions"`
}

func (a Account) HasPermission(
	code string,
) bool {
	for _, permission := range a.Permissions {
		if permission == code {
			return true
		}
	}

	return false
}

type Tokens struct {
	AccessToken string `json:"access_token"`

	RefreshToken string `json:"refresh_token"`

	TokenType string `json:"token_type"`

	AccessExpiresAt time.Time `json:"access_expires_at"`

	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type AuthResult struct {
	Staff  Account `json:"staff"`
	Tokens Tokens  `json:"tokens"`
}

type accountRecord struct {
	Account

	PasswordHash string
}

type refreshSessionRecord struct {
	ID string

	StaffAccountID string

	RefreshExpiresAt time.Time

	StaffStatus string
}
