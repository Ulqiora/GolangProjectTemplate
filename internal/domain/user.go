package domain

import (
	"encoding/json"
	"time"

	"GolangTemplateProject/internal/ports"
	"github.com/google/uuid"
)

type ID uuid.UUID

type User struct {
	Id        ID        `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	LastName  string    `json:"lastname" db:"lastname"`
	FirstName string    `json:"firstname" db:"firstname"`
	Login     string    `json:"login" db:"login"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (u *User) Params() map[string]interface{} {
	return map[string]interface{}{
		"email":      u.Email,
		"id":         u.Id,
		"lastname":   u.LastName,
		"firstname":  u.FirstName,
		"login":      u.Login,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}

func (u *User) Fields() []string {
	return []string{"id", "email", "hashed_password"}
}

func (u *User) PrimaryKey() any {
	return u.Id
}
func (u *User) Marshal() ([]byte, error) {
	return json.Marshal(u)
}

func (u *User) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, &u)
}

func (u *User) Scan(fields []string, scan ports.ScanFunc) error {
	err := scanner(map[string]any{
		"email":      &u.Email,
		"id":         &u.Id,
		"lastname":   &u.LastName,
		"firstname":  &u.FirstName,
		"login":      &u.Login,
		"created_at": &u.CreatedAt,
		"updated_at": &u.UpdatedAt,
	}, fields, scan)

	return err
}
