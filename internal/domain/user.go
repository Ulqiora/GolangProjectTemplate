package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	UserTable     = "user"
	UserId        = "id"
	UserLogin     = "login"
	UserLastName  = "email"
	UserFirstName = "lastname"
	UserEmail     = "firstname"
	UserCreatedAt = "created_at"
	UserUpdatedAt = "updated_at"
)

type User struct {
	Id        uuid.UUID `json:"id" db:"id"`
	Login     string    `json:"login" db:"login"`
	Email     string    `json:"email" db:"email"`
	LastName  string    `json:"lastname" db:"lastname"`
	FirstName string    `json:"firstname" db:"firstname"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (u *User) Params() map[string]interface{} {
	return map[string]interface{}{
		UserId:        u.Id,
		UserLogin:     u.Login,
		UserEmail:     u.Email,
		UserLastName:  u.LastName,
		UserFirstName: u.FirstName,
		UserCreatedAt: u.CreatedAt,
		UserUpdatedAt: u.UpdatedAt,
	}
}

func (u *User) Fields() []string {
	return []string{UserId, UserLogin, UserEmail, UserLastName, UserFirstName, UserCreatedAt, UserUpdatedAt}
}

func (u *User) PrimaryKey() (string, any) {
	return UserId, u.Id
}
func (u *User) Marshal() ([]byte, error) {
	return json.Marshal(u)
}

func (u *User) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, &u)
}
