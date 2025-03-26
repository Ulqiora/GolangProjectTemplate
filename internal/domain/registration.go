package domain

type RegistrationUserInfo struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegistrationUserResponse struct {
	UserUUID ID     `json:"user_uuid"`
	OtpUrl   string `json:"otp_url"`
}
