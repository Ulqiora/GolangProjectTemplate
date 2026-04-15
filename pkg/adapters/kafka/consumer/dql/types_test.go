package dql

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
)

func TestNewMapValues_InitializesMap(t *testing.T) {
	headers := []*sarama.RecordHeader{
		{Key: []byte("trace_id"), Value: []byte("abc-123")},
	}

	values := NewMapValues(headers)

	require.NotNil(t, values)
	require.Equal(t, "abc-123", (*values)["trace_id"])
}
