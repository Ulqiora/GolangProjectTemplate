package domain

import (
	"encoding/json"
	"time"
)

const VKUsersTopic = "users.vk.created"

type VKUserEvent struct {
	VKUserID   string    `json:"vk_user_id"`
	Login      string    `json:"login"`
	Email      string    `json:"email"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e VKUserEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *VKUserEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}
