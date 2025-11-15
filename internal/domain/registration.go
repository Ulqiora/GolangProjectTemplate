package domain

import "time"

type RegistrationUserInfo struct {
	Id        ID     `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Password  string `json:"password"`
}

type RegistrationUserResponse struct {
	UserId       ID        `json:"id"`
	OtpUrl       string    `json:"otp_url"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	CreatedAt    time.Time `json:"created_at"`
}
