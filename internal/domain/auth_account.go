package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	AuthUsersTable         = "auth_users"
	AuthUserIDField        = "id"
	AuthUserLoginField     = "login"
	AuthUserEmailField     = "email"
	AuthUserFirstNameField = "first_name"
	AuthUserLastNameField  = "last_name"
	AuthUserProviderField  = "provider"
	AuthUserCreatedAtField = "created_at"
	AuthUserUpdatedAtField = "updated_at"
)

type AuthProvider string

const (
	AuthProviderPassword AuthProvider = "password"
	AuthProviderYandexID AuthProvider = "yandex_id"
	AuthProviderVK       AuthProvider = "vk"
)

type UserAccount struct {
	ID        uuid.UUID    `json:"id" db:"id"`
	Login     string       `json:"login" db:"login"`
	Email     string       `json:"email" db:"email"`
	FirstName string       `json:"first_name" db:"first_name"`
	LastName  string       `json:"last_name" db:"last_name"`
	Provider  AuthProvider `json:"provider" db:"provider"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
}

func (u UserAccount) Params() map[string]interface{} {
	return map[string]interface{}{
		AuthUserIDField:        u.ID,
		AuthUserLoginField:     u.Login,
		AuthUserEmailField:     u.Email,
		AuthUserFirstNameField: u.FirstName,
		AuthUserLastNameField:  u.LastName,
		AuthUserProviderField:  string(u.Provider),
		AuthUserCreatedAtField: u.CreatedAt,
		AuthUserUpdatedAtField: u.UpdatedAt,
	}
}

func (u UserAccount) Fields() []string {
	return []string{
		AuthUserIDField,
		AuthUserLoginField,
		AuthUserEmailField,
		AuthUserFirstNameField,
		AuthUserLastNameField,
		AuthUserProviderField,
		AuthUserCreatedAtField,
		AuthUserUpdatedAtField,
	}
}

func (u UserAccount) PrimaryKey() (string, any) {
	return AuthUserIDField, u.ID
}

func (u *UserAccount) Marshal() ([]byte, error) {
	return json.Marshal(u)
}

func (u *UserAccount) Unmarshal(data []byte) error {
	return json.Unmarshal(data, u)
}
