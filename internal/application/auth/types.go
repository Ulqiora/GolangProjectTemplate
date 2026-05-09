package auth

import "time"

type RegisterInput struct {
	Login     string
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type LoginInput struct {
	Identifier string
	Password   string
}

type CompleteYandexInput struct {
	Code  string
	State string
}

type AuthResult struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	Provider     string
	CreatedAt    time.Time
}
