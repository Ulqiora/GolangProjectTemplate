package domain

import (
	"encoding/json"
	"time"
)

const UserCreatedTopic = "users.created"

type UserCreatedEvent struct {
	UserID     string       `json:"user_id"`
	Provider   AuthProvider `json:"provider"`
	OccurredAt time.Time    `json:"occurred_at"`
}

func (e UserCreatedEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func NewUserCreatedOutboxMessage(account *UserAccount) (*OutboxMessage, error) {
	now := time.Now().UTC()
	payload, err := UserCreatedEvent{
		UserID:     account.ID.String(),
		Provider:   account.Provider,
		OccurredAt: now,
	}.Marshal()
	if err != nil {
		return nil, err
	}

	return &OutboxMessage{
		ID:           NewUUIDv7(),
		Topic:        UserCreatedTopic,
		MessageKey:   account.ID.String(),
		Payload:      payload,
		Status:       OutboxStatusPending,
		AttemptCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}
