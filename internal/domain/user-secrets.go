package domain

import (
	"encoding/json"
	"time"

	"GolangTemplateProject/internal/ports"
	"github.com/google/uuid"
)

type UserSecrets struct {
	UserId         uuid.UUID `json:"userId" db:"user_id"`
	HashedPassword string    `json:"hashed_password" db:"hashed_password"`
	OtpSecret      string    `json:"otp_secret" db:"otp_secret"`
	OtpUrl         string    `json:"otp_url" db:"otp_url"`
	OtpNonce       string    `json:"otp_nonce" db:"otp_nonce"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

func (u *UserSecrets) Params() map[string]interface{} {
	return map[string]interface{}{
		"user_id":         u.UserId,
		"hashed_password": u.HashedPassword,
		"otp_secret":      u.OtpSecret,
		"otp_url":         u.OtpUrl,
		"otp_nonce":       u.OtpNonce,
		"updated_at":      u.UpdatedAt,
	}
}

func (u *UserSecrets) Fields() []string {
	return []string{"id", "email", "hashed_password"}
}

func (u *UserSecrets) PrimaryKey() any {
	return u.UserId
}
func (u *UserSecrets) Marshal() ([]byte, error) {
	return json.Marshal(u)
}

func (u *UserSecrets) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, &u)
}

func (u *UserSecrets) Scan(fields []string, scan ports.ScanFunc) error {
	err := scanner(map[string]any{
		"user_id":         &u.UserId,
		"hashed_password": &u.HashedPassword,
		"otp_secret":      &u.OtpSecret,
		"otp_url":         &u.OtpUrl,
		"otp_nonce":       &u.OtpNonce,
		"updated_at":      &u.UpdatedAt,
	}, fields, scan)

	return err
}
