package customer

import "time"

type Customer struct {
	ID        string    `json:"id"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email,omitempty"`
	FullName  string    `json:"full_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	AvatarURL          string     `json:"avatar_url,omitempty"`
	AvatarURLExpiresAt *time.Time `json:"avatar_url_expires_at,omitempty"`
	AvatarUpdatedAt    *time.Time `json:"avatar_updated_at,omitempty"`
}

type Address struct {
	ID            string    `json:"id"`
	CustomerID    string    `json:"customer_id"`
	Label         string    `json:"label"`
	RecipientName string    `json:"recipient_name"`
	Phone         string    `json:"phone"`
	AddressLine1  string    `json:"address_line1"`
	AddressLine2  string    `json:"address_line2,omitempty"`
	City          string    `json:"city"`
	Area          string    `json:"area"`
	PostalCode    string    `json:"postal_code,omitempty"`
	IsDefault     bool      `json:"is_default"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
