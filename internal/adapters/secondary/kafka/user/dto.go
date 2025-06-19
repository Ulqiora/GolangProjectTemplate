package user

import "encoding/json"

type UserDTO struct {
	Id             int    `json:"id"`
	Email          string `json:"email"`
	HashedPassword string `json:"password"`
}

func (u UserDTO) Marshal() ([]byte, error) {
	return json.Marshal(u)
}

func (u UserDTO) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, &u)
}
