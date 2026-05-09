package kafka

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/pkg/adapters/kafka/producer"
)

const userCreatedAvroSchema = `{"type":"record","name":"UserCreated","namespace":"auth.v1","fields":[{"name":"user_id","type":"string"},{"name":"provider","type":"string"},{"name":"occurred_at","type":{"type":"long","logicalType":"timestamp-millis"}}]}`

type schemaRegistryUserCreatedEncoder struct {
	registryURL string
	subject     string
	httpClient  *http.Client

	mu       sync.Mutex
	schemaID int
}

func newOutboxSerializer(cfg producer.Config) producer.Serializer[[]byte] {
	if strings.TrimSpace(cfg.SchemaRegistry.URL) == "" {
		return func(value []byte) ([]byte, error) {
			return value, nil
		}
	}

	subject := strings.TrimSpace(cfg.SchemaRegistry.Subject)
	if subject == "" {
		subject = cfg.Topic + "-value"
	}
	encoder := &schemaRegistryUserCreatedEncoder{
		registryURL: strings.TrimRight(strings.TrimSpace(cfg.SchemaRegistry.URL), "/"),
		subject:     subject,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}

	return func(value []byte) ([]byte, error) {
		var event domain.UserCreatedEvent
		if err := json.Unmarshal(value, &event); err != nil {
			return nil, fmt.Errorf("decode user created event: %w", err)
		}
		return encoder.Encode(event)
	}
}

func (e *schemaRegistryUserCreatedEncoder) Encode(event domain.UserCreatedEvent) ([]byte, error) {
	schemaID, err := e.ensureSchemaID()
	if err != nil {
		return nil, err
	}

	var payload bytes.Buffer
	payload.WriteByte(0)
	if err := binary.Write(&payload, binary.BigEndian, uint32(schemaID)); err != nil {
		return nil, fmt.Errorf("write schema id: %w", err)
	}

	writeAvroString(&payload, event.UserID)
	writeAvroString(&payload, string(event.Provider))
	writeAvroLong(&payload, event.OccurredAt.UTC().UnixMilli())
	return payload.Bytes(), nil
}

func (e *schemaRegistryUserCreatedEncoder) ensureSchemaID() (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.schemaID != 0 {
		return e.schemaID, nil
	}

	requestBody, err := json.Marshal(struct {
		SchemaType string `json:"schemaType"`
		Schema     string `json:"schema"`
	}{
		SchemaType: "AVRO",
		Schema:     userCreatedAvroSchema,
	})
	if err != nil {
		return 0, fmt.Errorf("build schema registry request: %w", err)
	}

	endpoint := e.registryURL + "/subjects/" + url.PathEscape(e.subject) + "/versions"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return 0, fmt.Errorf("build schema registry http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("register user created schema: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read schema registry response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("register user created schema: status=%d body=%s", resp.StatusCode, string(body))
	}

	var response struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("decode schema registry response: %w", err)
	}
	if response.ID <= 0 {
		return 0, fmt.Errorf("schema registry returned invalid schema id: %d", response.ID)
	}

	e.schemaID = response.ID
	return e.schemaID, nil
}

func writeAvroString(buf *bytes.Buffer, value string) {
	writeAvroLong(buf, int64(len(value)))
	buf.WriteString(value)
}

func writeAvroLong(buf *bytes.Buffer, value int64) {
	encoded := uint64(value<<1) ^ uint64(value>>63)
	for encoded >= 0x80 {
		buf.WriteByte(byte(encoded) | 0x80)
		encoded >>= 7
	}
	buf.WriteByte(byte(encoded))
}
