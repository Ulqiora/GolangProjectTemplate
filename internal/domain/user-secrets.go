package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	UserSecretTable          = "UserSecret"
	UserSecretUserId         = "user_id"
	UserSecretHashedPassword = "hashed_password"
	UserSecretOtpSecret      = "email"
	UserSecretOtpUrl         = "lastname"
	UserSecretNonce          = "firstname"
	UserSecretUpdatedAt      = "created_at"
)

type UserSecrets struct {
	UserId         uuid.UUID `json:"user_id" db:"user_id"`
	HashedPassword string    `json:"hashed_password" db:"hashed_password"`
	OtpSecret      string    `json:"otp_secret" db:"otp_secret"`
	OtpUrl         string    `json:"otp_url" db:"otp_url"`
	OtpNonce       string    `json:"otp_nonce" db:"otp_nonce"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

func (u *UserSecrets) Params() map[string]interface{} {
	return map[string]interface{}{
		UserSecretUserId:         u.UserId,
		UserSecretHashedPassword: u.HashedPassword,
		UserSecretOtpSecret:      u.OtpSecret,
		UserSecretOtpUrl:         u.OtpUrl,
		UserSecretNonce:          u.OtpNonce,
		UserSecretUpdatedAt:      u.UpdatedAt,
	}
}

func (u *UserSecrets) Fields() []string {
	return []string{
		UserSecretUserId,
		UserSecretHashedPassword,
		UserSecretOtpSecret,
		UserSecretOtpUrl,
		UserSecretNonce,
		UserSecretUpdatedAt,
	}
}

func (u *UserSecrets) PrimaryKey() (string, any) {
	return UserId, u.UserId
}
func (u *UserSecrets) Marshal() ([]byte, error) {
	return json.Marshal(u)
}

func (u *UserSecrets) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, &u)
}
