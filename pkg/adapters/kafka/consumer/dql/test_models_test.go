package dql

import "encoding/json"

type testMessage struct {
	Value string `json:"value"`
}

type nonPointerMessage struct {
	Value string `json:"value"`
}

func (m *testMessage) Params() map[string]interface{} {
	return map[string]interface{}{"value": m.Value}
}

func (m *testMessage) Fields() []string {
	return []string{"value"}
}

func (m *testMessage) PrimaryKey() (string, any) {
	return "value", m.Value
}

func (m *testMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *testMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}

func (m nonPointerMessage) Params() map[string]interface{} {
	return map[string]interface{}{"value": m.Value}
}

func (m nonPointerMessage) Fields() []string {
	return []string{"value"}
}

func (m nonPointerMessage) PrimaryKey() (string, any) {
	return "value", m.Value
}

func (m nonPointerMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m nonPointerMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &m)
}
