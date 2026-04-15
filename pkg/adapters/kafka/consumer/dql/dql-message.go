package dql

import (
	"encoding/json"
	"time"
)

const (
	DlqMessageTable                       = "dlq_message"
	DlqMessageId                          = "id"
	DlqMessageTopic                       = "topic"
	DlqMessagePartition                   = "partition"
	DlqMessageOffset                      = "message_offset"
	DlqMessageObjectIndex                 = "object_index"
	DlqMessagePayload                     = "payload"
	DlqMessageAttemptNumber               = "attempt_number"
	DlqMessageLastAttemptError            = "last_attempt_error"
	DlqMessageLastAttemptErrorDescription = "last_attempt_error_description"
	DlqMessageLastAttemptTime             = "last_attempt_time"
	DlqMessageDeleted                     = "deleted"
	DlqMessageCreatedAt                   = "created_at"
	DlqMessageUpdatedAt                   = "updated_at"
)

type DLQMessage struct {
	ID                          string    `json:"id" db:"id"`
	Topic                       string    `json:"topic" db:"topic"`
	Partition                   int32     `json:"partition" db:"partition"`
	Offset                      int64     `json:"offset" db:"message_offset"`
	ObjectIndex                 int64     `json:"object_index" db:"object_index"`
	Payload                     string    `json:"payload" db:"payload"`
	Status                      string    `json:"status" db:"status"`
	AttemptNumber               int32     `json:"attempt_number" db:"attempt_number"`
	LastAttemptError            string    `json:"last_attempt_error" db:"last_attempt_error"`
	LastAttemptErrorDescription string    `json:"last_attempt_error_description" db:"last_attempt_error_description"`
	LastAttemptTime             time.Time `json:"last_attempt_time" db:"last_attempt_time"`
	Deleted                     bool      `json:"deleted" db:"deleted"`
	CreatedAt                   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                   time.Time `json:"updated_at" db:"updated_at"`
}

func (u *DLQMessage) Params() map[string]interface{} {
	return map[string]interface{}{
		DlqMessageId:                          u.ID,
		DlqMessageTopic:                       u.Topic,
		DlqMessagePartition:                   u.Partition,
		DlqMessageOffset:                      u.Offset,
		DlqMessageObjectIndex:                 u.ObjectIndex,
		DlqMessagePayload:                     u.Payload,
		DlqMessageAttemptNumber:               u.AttemptNumber,
		DlqMessageLastAttemptError:            u.LastAttemptError,
		DlqMessageLastAttemptErrorDescription: u.LastAttemptErrorDescription,
		DlqMessageLastAttemptTime:             u.LastAttemptTime,
		DlqMessageCreatedAt:                   u.CreatedAt,
		DlqMessageUpdatedAt:                   u.UpdatedAt,
	}
}

func (u *DLQMessage) Fields() []string {
	return []string{
		DlqMessageId,
		DlqMessageTopic,
		DlqMessagePartition,
		DlqMessageOffset,
		DlqMessageObjectIndex,
		DlqMessagePayload,
		DlqMessageAttemptNumber,
		DlqMessageLastAttemptError,
		DlqMessageLastAttemptErrorDescription,
		DlqMessageLastAttemptTime,
		DlqMessageDeleted,
		DlqMessageCreatedAt,
		DlqMessageUpdatedAt,
	}
}

func (u *DLQMessage) PrimaryKey() (string, any) {
	return DlqMessageId, u.ID
}
func (u *DLQMessage) Marshal() ([]byte, error) {
	return json.Marshal(u)
}

func (u *DLQMessage) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, &u)
}
