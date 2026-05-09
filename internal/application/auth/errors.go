package auth

import "errors"

var (
	ErrLoginRequired           = errors.New("login is required")
	ErrEmailRequired           = errors.New("email is required")
	ErrPasswordRequired        = errors.New("password is required")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrUserAlreadyExists       = errors.New("user already exists")
	ErrYandexCodeRequired      = errors.New("yandex authorization code is required")
	ErrYandexStateRequired     = errors.New("yandex state is required")
	ErrFirstNameRequired       = errors.New("first name is required")
	ErrLastNameRequired        = errors.New("last name is required")
	ErrIdentifierRequired      = errors.New("identifier is required")
	ErrUnsupportedAuthState    = errors.New("unsupported auth state")
	ErrYandexProfileIncomplete = errors.New("yandex profile does not contain enough information")
)
