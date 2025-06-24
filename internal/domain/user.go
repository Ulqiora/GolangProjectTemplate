package domain

import (
	"encoding/json"

	"GolangTemplateProject/internal/ports"
	"github.com/google/uuid"
)

type ID uuid.UUID

type User struct {
	Id             ID     `json:"id" db:"id"`
	Email          string `json:"email" db:"email"`
	HashedPassword string `json:"hashed_password" db:"hashed_password"`
}

func (u User) Params() map[string]interface{} {
	return map[string]interface{}{
		"email":           u.Email,
		"hashed_password": u.HashedPassword,
		"id":              u.Id,
	}
}

func (u User) Fields() []string {
	return []string{"id", "email", "hashed_password"}
}

func (u User) PrimaryKey() any {
	return u.Id
}
func (u User) Marshal() ([]byte, error) {
	return json.Marshal(u)
}

func (u User) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, &u)
}

func (u User) Scan(fields []string, scan ports.ScanFunc) error {
	err := scanner(map[string]any{
		"email":           &u.Email,
		"hashed_password": &u.HashedPassword,
		"id":              &u.Id,
	}, fields, scan)

	return err
}
