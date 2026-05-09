package domain

import "github.com/google/uuid"

func NewUUIDv7() uuid.UUID {
	value, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}

	return value
}
