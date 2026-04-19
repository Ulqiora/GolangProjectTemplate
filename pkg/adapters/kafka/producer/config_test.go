package producer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildProduceConfig_ValidateInput(t *testing.T) {
	_, err := BuildProduceConfig(Config{})
	require.ErrorIs(t, err, ErrTopicRequired)

	_, err = BuildProduceConfig(Config{Topic: "topic"})
	require.ErrorIs(t, err, ErrBrokersRequired)

	_, err = BuildProduceConfig(Config{
		Topic:           "topic",
		Brokers:         []string{"localhost:9092"},
		CompressionType: 127,
		ProduceSettings: ProduceSettings{RequiredAcks: -1},
	})
	require.ErrorIs(t, err, ErrUnsupportedCompression)

	_, err = BuildProduceConfig(Config{
		Topic:           "topic",
		Brokers:         []string{"localhost:9092"},
		CompressionType: 0,
		ProduceSettings: ProduceSettings{RequiredAcks: 7},
	})
	require.ErrorIs(t, err, ErrUnsupportedRequiredAcks)
}
