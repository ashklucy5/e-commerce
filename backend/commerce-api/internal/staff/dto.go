package staff

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
