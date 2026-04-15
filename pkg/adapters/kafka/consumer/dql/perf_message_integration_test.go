//go:build integration
// +build integration

package dql

import "encoding/json"

type perfMessage struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

func (m *perfMessage) Params() map[string]interface{} {
	return map[string]interface{}{"id": m.ID, "value": m.Value}
}

func (m *perfMessage) Fields() []string {
	return []string{"id", "value"}
}

func (m *perfMessage) PrimaryKey() (string, any) {
	return "id", m.ID
}

func (m *perfMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *perfMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}
