package kafka

import (
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"GolangTemplateProject/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestSchemaRegistryUserCreatedEncoderRegistersSchemaAndEncodesConfluentAvro(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/subjects/users.created-value/versions", r.URL.Path)

		var body struct {
			SchemaType string `json:"schemaType"`
			Schema     string `json:"schema"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "AVRO", body.SchemaType)
		require.JSONEq(t, userCreatedAvroSchema, body.Schema)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":42}`))
	}))
	defer server.Close()

	encoder := &schemaRegistryUserCreatedEncoder{
		registryURL: server.URL,
		subject:     "users.created-value",
		httpClient:  server.Client(),
	}
	event := domain.UserCreatedEvent{
		UserID:     "user-1",
		Provider:   domain.AuthProviderPassword,
		OccurredAt: time.UnixMilli(1715000000123).UTC(),
	}

	payload, err := encoder.Encode(event)
	require.NoError(t, err)
	require.Len(t, payload, 5+1+len("user-1")+1+len("password")+6)
	require.Equal(t, byte(0), payload[0])
	require.Equal(t, uint32(42), binary.BigEndian.Uint32(payload[1:5]))
	require.Equal(t, []byte{12}, payload[5:6])
	require.Equal(t, "user-1", string(payload[6:12]))
	require.Equal(t, []byte{16}, payload[12:13])
	require.Equal(t, "password", string(payload[13:21]))
	require.Equal(t, []byte{246, 249, 185, 223, 233, 99}, payload[21:])

	_, err = encoder.Encode(event)
	require.NoError(t, err)
	require.Equal(t, 1, requests)
}
