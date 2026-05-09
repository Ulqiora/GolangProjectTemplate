package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	ExternalIdentitiesTable        = "auth_external_identities"
	ExternalIdentityIDField        = "id"
	ExternalIdentityUserIDField    = "user_id"
	ExternalIdentityProviderField  = "provider"
	ExternalIdentitySubjectField   = "subject"
	ExternalIdentityCreatedAtField = "created_at"
	ExternalIdentityUpdatedAtField = "updated_at"
)

type ExternalIdentity struct {
	ID        uuid.UUID    `json:"id" db:"id"`
	UserID    uuid.UUID    `json:"user_id" db:"user_id"`
	Provider  AuthProvider `json:"provider" db:"provider"`
	Subject   string       `json:"subject" db:"subject"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
}

func (i ExternalIdentity) Params() map[string]interface{} {
	return map[string]interface{}{
		ExternalIdentityIDField:        i.ID,
		ExternalIdentityUserIDField:    i.UserID,
		ExternalIdentityProviderField:  string(i.Provider),
		ExternalIdentitySubjectField:   i.Subject,
		ExternalIdentityCreatedAtField: i.CreatedAt,
		ExternalIdentityUpdatedAtField: i.UpdatedAt,
	}
}

func (i ExternalIdentity) Fields() []string {
	return []string{
		ExternalIdentityIDField,
		ExternalIdentityUserIDField,
		ExternalIdentityProviderField,
		ExternalIdentitySubjectField,
		ExternalIdentityCreatedAtField,
		ExternalIdentityUpdatedAtField,
	}
}

func (i ExternalIdentity) PrimaryKey() (string, any) {
	return ExternalIdentityIDField, i.ID
}

func (i *ExternalIdentity) Marshal() ([]byte, error) {
	return json.Marshal(i)
}

func (i *ExternalIdentity) Unmarshal(data []byte) error {
	return json.Unmarshal(data, i)
}
