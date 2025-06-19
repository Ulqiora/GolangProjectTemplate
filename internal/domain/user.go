package domain

import "encoding/json"

type ID string

type User struct {
	Id             ID     `json:"id"`
	Email          string `json:"email"`
	HashedPassword string `json:"password"`
}

func (u User) Params() map[string]interface{} {
	return map[string]interface{}{
		"email":    u.Email,
		"password": u.HashedPassword,
		"id":       u.Id,
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
