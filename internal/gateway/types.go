package gateway

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type passwordResetRequestRequest struct {
	Email string `json:"email"`
}

type passwordResetConfirmationRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"password"`
}
