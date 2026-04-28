package kafka_test

import (
	"context"
	"sync"
	"testing"

	. "GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/mocks"
)

func TestKafkaMocksSatisfyInterfaces(t *testing.T) {
	t.Parallel()

	var _ Runnable = (*mocks.Runnable)(nil)
	var _ TypedProducer[string] = (*mocks.TypedProducer[string])(nil)
	var _ ClosableTypedProducer[string] = (*mocks.TypedProducer[string])(nil)
	var _ TransactionalProducer[string] = (*mocks.TransactionalProducer[string])(nil)
	var _ Consumer = (*mocks.Consumer)(nil)
	var _ ClosableConsumer = (*mocks.Consumer)(nil)

	producer := &mocks.TypedProducer[string]{Topic: "topic"}
	if producer.TopicName() != "topic" {
		t.Fatal("unexpected topic")
	}
	if err := producer.Run(context.Background(), &sync.WaitGroup{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}
