package dql

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"GolangTemplateProject/pkg/logger/attribute"
	"github.com/IBM/sarama"
)

func messageAttributes(message *sarama.ConsumerMessage) []attribute.Field {
	return []attribute.Field{
		attribute.String("topic", message.Topic),
		attribute.Int("partition", int(message.Partition)),
		attribute.Int64("offset", message.Offset),
		attribute.Int("payload_bytes", len(message.Value)),
		attribute.Int("headers_count", len(message.Headers)),
		attribute.String("key", string(message.Key)),
		attribute.Int64("timestamp_unix", message.Timestamp.Unix()),
	}
}

func processingAttributes(startedAt time.Time, status string) []attribute.Field {
	return []attribute.Field{
		attribute.String("status", status),
		attribute.Float64("duration_ms", float64(time.Since(startedAt).Milliseconds())),
	}
}

func claimsAttributes(claims map[string][]int32) []attribute.Field {
	partitionsTotal := 0
	topics := make([]string, 0, len(claims))
	for topic, partitions := range claims {
		topics = append(topics, fmt.Sprintf("%s=%v", topic, partitions))
		partitionsTotal += len(partitions)
	}
	sort.Strings(topics)

	return []attribute.Field{
		attribute.Int("claims_topics_count", len(claims)),
		attribute.Int("claims_partitions_count", partitionsTotal),
		attribute.String("claims", strings.Join(topics, ",")),
	}
}
