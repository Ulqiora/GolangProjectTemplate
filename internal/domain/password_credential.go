package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	PasswordCredentialsTable         = "auth_password_credentials"
	PasswordCredentialUserIDField    = "user_id"
	PasswordCredentialHashField      = "password_hash"
	PasswordCredentialCreatedAtField = "created_at"
	PasswordCredentialUpdatedAtField = "updated_at"
)

type PasswordCredential struct {
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	PasswordHash string    `json:"password_hash" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

func (c PasswordCredential) Params() map[string]interface{} {
	return map[string]interface{}{
		PasswordCredentialUserIDField:    c.UserID,
		PasswordCredentialHashField:      c.PasswordHash,
		PasswordCredentialCreatedAtField: c.CreatedAt,
		PasswordCredentialUpdatedAtField: c.UpdatedAt,
	}
}

func (c PasswordCredential) Fields() []string {
	return []string{
		PasswordCredentialUserIDField,
		PasswordCredentialHashField,
		PasswordCredentialCreatedAtField,
		PasswordCredentialUpdatedAtField,
	}
}

func (c PasswordCredential) PrimaryKey() (string, any) {
	return PasswordCredentialUserIDField, c.UserID
}

func (c *PasswordCredential) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

func (c *PasswordCredential) Unmarshal(data []byte) error {
	return json.Unmarshal(data, c)
}
