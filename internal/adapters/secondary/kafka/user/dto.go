package user

import "encoding/json"

type UserDTO struct {
	Id        int    `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Username  string `json:"username"`
	Login     string `json:"login"`
}

func (u UserDTO) Marshal() ([]byte, error) {
	return json.Marshal(u)
}

func (u UserDTO) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, &u)
}
