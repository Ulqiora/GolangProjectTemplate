package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/IBM/sarama"
)

type MessageObject interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

type MapValues map[string]string

type ExecDomainFunc[T MessageObject] func(ctx context.Context, object T, values MapValues) error

type TopicConsumerGroup[M MessageObject] struct {
	groupID        string
	topic          string
	producerSamara sarama.ConsumerGroup
	logger         sarama.StdLogger
	base           *topicConsumerGroupBase[M]
}

func NewTopicConsumerGroup[M MessageObject](config Config, logger sarama.StdLogger, domainFunc ExecDomainFunc[M]) (*TopicConsumerGroup[M], error) {
	saramaConfig := sarama.NewConfig()
	if val, ok := offsetMap[config.GroupSettings.OffsetInitial]; ok {
		saramaConfig.Consumer.Offsets.Initial = val
	} else {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	}
	if val, ok := rebalanceStrategyMap[config.GroupSettings.RebalancedGroupStrategy]; ok {
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{val}
	} else {
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	}
	saramaConfig.Consumer.Return.Errors = config.GroupSettings.ReturnErrors
	if val, ok := isolationLevelMap[config.GroupSettings.IsolationLevel]; ok {
		saramaConfig.Consumer.IsolationLevel = val
	} else {
		saramaConfig.Consumer.IsolationLevel = sarama.ReadUncommitted
	}
	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, config.GroupSettings.GroupID, saramaConfig)

	return &TopicConsumerGroup[M]{
		groupID:        config.GroupSettings.GroupID,
		logger:         logger,
		producerSamara: consumerGroup,
		topic:          config.Topic,
		base:           newTopicConsumerGroupBase[M](logger, domainFunc),
	}, err
}

func (t *TopicConsumerGroup[T]) WaitStoppedSession() <-chan struct{} {
	return t.base.ready
}

func (t *TopicConsumerGroup[T]) ResumePartitions() {
	t.producerSamara.ResumeAll()
}

func (t *TopicConsumerGroup[T]) GroupID() string {
	return t.groupID
}

func (t *TopicConsumerGroup[T]) Run(ctx context.Context) {
	go func() {
		for {
			err := t.producerSamara.Consume(ctx, []string{t.topic}, t.base)
			if errors.Is(err, sarama.ErrClosedConsumerGroup) {
				return
			}
			t.logger.Println("Error on consumer group", err)
			if ctx.Err() != nil {
				t.logger.Println("Context of kafka consumer reset by signal", err)
				return
			}
			t.logger.Println("Restarting kafka consumer group", t.groupID)
			t.base.ready = make(chan struct{})
		}
	}()
}

type topicConsumerGroupBase[T MessageObject] struct {
	ready        chan struct{}
	logger       sarama.StdLogger
	execFunction ExecDomainFunc[T]
}

func newTopicConsumerGroupBase[T MessageObject](logger sarama.StdLogger, domainFunc ExecDomainFunc[T]) *topicConsumerGroupBase[T] {
	return &topicConsumerGroupBase[T]{
		ready:        make(chan struct{}),
		logger:       logger,
		execFunction: domainFunc,
	}
}

func (t *topicConsumerGroupBase[T]) Setup(sarama.ConsumerGroupSession) error {
	close(t.ready)
	return nil
}

func (t *topicConsumerGroupBase[T]) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (t *topicConsumerGroupBase[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				t.logger.Println("message channel closed")
				return nil
			}
			t.logger.Printf("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
			var object T
			err := json.Unmarshal(message.Value, &object)
			if err != nil {
				t.logger.Printf("Error on unmarshalling message: %s\n", err)
			}
			if err = t.execFunction(session.Context(), object, NewMapValues(message.Headers)); err == nil {
				session.MarkMessage(message, "")
			}
		case <-session.Context().Done():
			t.logger.Print("session canceled")
			return nil
		default:
			<-time.After(time.Second)
		}
	}
}
