package customer

type UpdateProfileRequest struct {
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	FullName *string `json:"full_name"`
}

type CreateAddressRequest struct {
	Label         string `json:"label"`
	RecipientName string `json:"recipient_name"`
	Phone         string `json:"phone"`
	AddressLine1  string `json:"address_line1"`
	AddressLine2  string `json:"address_line2"`
	City          string `json:"city"`
	Area          string `json:"area"`
	PostalCode    string `json:"postal_code"`
	IsDefault     bool   `json:"is_default"`
}

type UpdateAddressRequest struct {
	Label         *string `json:"label"`
	RecipientName *string `json:"recipient_name"`
	Phone         *string `json:"phone"`
	AddressLine1  *string `json:"address_line1"`
	AddressLine2  *string `json:"address_line2"`
	City          *string `json:"city"`
	Area          *string `json:"area"`
	PostalCode    *string `json:"postal_code"`
	IsDefault     *bool   `json:"is_default"`
}
