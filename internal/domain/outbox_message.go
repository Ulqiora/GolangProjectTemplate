package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	OutboxMessagesTable             = "auth_outbox_messages"
	OutboxMessageIDField            = "id"
	OutboxMessageTopicField         = "topic"
	OutboxMessageKeyField           = "message_key"
	OutboxMessagePayloadField       = "payload"
	OutboxMessageStatusField        = "status"
	OutboxMessageAttemptCountField  = "attempt_count"
	OutboxMessageLastErrorField     = "last_error"
	OutboxMessagePublishedAtField   = "published_at"
	OutboxMessageNextAttemptAtField = "next_attempt_at"
	OutboxMessageCreatedAtField     = "created_at"
	OutboxMessageUpdatedAtField     = "updated_at"
	OutboxStatusPending             = "pending"
	OutboxStatusProcessing          = "processing"
	OutboxStatusSent                = "sent"
	OutboxStatusFailed              = "failed"
)

type OutboxMessage struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Topic         string     `json:"topic" db:"topic"`
	MessageKey    string     `json:"message_key" db:"message_key"`
	Payload       []byte     `json:"payload" db:"payload"`
	Status        string     `json:"status" db:"status"`
	AttemptCount  int32      `json:"attempt_count" db:"attempt_count"`
	LastError     string     `json:"last_error" db:"last_error"`
	PublishedAt   *time.Time `json:"published_at" db:"published_at"`
	NextAttemptAt *time.Time `json:"next_attempt_at" db:"next_attempt_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

func (m OutboxMessage) Params() map[string]interface{} {
	return map[string]interface{}{
		OutboxMessageIDField:            m.ID,
		OutboxMessageTopicField:         m.Topic,
		OutboxMessageKeyField:           m.MessageKey,
		OutboxMessagePayloadField:       m.Payload,
		OutboxMessageStatusField:        m.Status,
		OutboxMessageAttemptCountField:  m.AttemptCount,
		OutboxMessageLastErrorField:     m.LastError,
		OutboxMessagePublishedAtField:   m.PublishedAt,
		OutboxMessageNextAttemptAtField: m.NextAttemptAt,
		OutboxMessageCreatedAtField:     m.CreatedAt,
		OutboxMessageUpdatedAtField:     m.UpdatedAt,
	}
}

func (m OutboxMessage) Fields() []string {
	return []string{
		OutboxMessageIDField,
		OutboxMessageTopicField,
		OutboxMessageKeyField,
		OutboxMessagePayloadField,
		OutboxMessageStatusField,
		OutboxMessageAttemptCountField,
		OutboxMessageLastErrorField,
		OutboxMessagePublishedAtField,
		OutboxMessageNextAttemptAtField,
		OutboxMessageCreatedAtField,
		OutboxMessageUpdatedAtField,
	}
}

func (m OutboxMessage) PrimaryKey() (string, any) {
	return OutboxMessageIDField, m.ID
}

func (m *OutboxMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *OutboxMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}
