package domain

import (
	"time"

	"github.com/google/uuid"
)

type RegistrationUserInfo struct {
	Id        uuid.UUID `json:"id"`
	Login     string    `json:"login"`
	Email     string    `json:"email"`
	Firstname string    `json:"firstname"`
	Lastname  string    `json:"lastname"`
	Password  string    `json:"password"`
}

type RegistrationUserResponse struct {
	UserId       uuid.UUID `json:"id"`
	OtpUrl       string    `json:"otp_url"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	CreatedAt    time.Time `json:"created_at"`
}
