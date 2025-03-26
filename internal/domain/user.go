package domain

type ID string

type User struct {
	Id             ID     `json:"id"`
	Email          string `json:"email"`
	HashedPassword string `json:"password"`
}
