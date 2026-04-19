package dql

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type ConsumerDQL[T BaseDqlConsumerModel] struct {
	groupID        string
	topic          string
	producerSamara sarama.ConsumerGroup
	logger         logger.Logger
	metrics        *consumerMetrics
	stopped        chan struct{}
	stopOnce       sync.Once
	base           *baseTopicConsumerGroup[T]
}

const consumeRestartBackoff = time.Second

func (t *ConsumerDQL[T]) WaitStoppedSession() <-chan struct{} {
	return t.stopped
}

func (t *ConsumerDQL[T]) ResumePartitions() {
	t.producerSamara.ResumeAll()
}

func (t *ConsumerDQL[T]) GroupID() string {
	return t.groupID
}

func (t *ConsumerDQL[T]) Run(ctx context.Context) {
	go func() {
		defer t.stopOnce.Do(func() {
			if err := t.producerSamara.Close(); err != nil && !errors.Is(err, sarama.ErrClosedConsumerGroup) {
				t.logger.Error(
					LogConsumerGroupCloseFailed,
					attribute.String("topic", t.topic),
					attribute.String("group_id", t.groupID),
					attribute.String("error", err.Error()),
				)
			} else {
				t.logger.Info(LogConsumerGroupClosed, attribute.String("topic", t.topic), attribute.String("group_id", t.groupID))
			}
			t.logger.Info(LogConsumerLoopStopped, attribute.String("topic", t.topic))
			close(t.stopped)
		})

		t.logger.Info(
			LogConsumerLoopStarted,
			attribute.String("topic", t.topic),
			attribute.String("group_id", t.groupID),
		)

		for {
			err := t.producerSamara.Consume(ctx, []string{t.topic}, t.base)
			if errors.Is(err, sarama.ErrClosedConsumerGroup) || ctx.Err() != nil {
				if ctx.Err() != nil {
					t.metrics.consumeRestarts.WithLabelValues(t.groupID, t.topic, "context_cancelled").Inc()
					t.logger.Info(
						LogConsumerStoppedByContext,
						attribute.String("group_id", t.groupID),
						attribute.String("topic", t.topic),
						attribute.String("reason", ctx.Err().Error()),
					)
				}
				return
			}
			if err == nil {
				t.metrics.consumeRestarts.WithLabelValues(t.groupID, t.topic, "session_finished").Inc()
				t.logger.Info(
					LogConsumerSessionFinished,
					attribute.String("group_id", t.groupID),
					attribute.String("topic", t.topic),
				)
				continue
			}

			t.metrics.consumeRestarts.WithLabelValues(t.groupID, t.topic, "consume_error").Inc()
			t.logger.Error(
				LogConsumerReturnedError,
				attribute.String("group_id", t.groupID),
				attribute.String("topic", t.topic),
				attribute.String("error", err.Error()),
			)
			t.logger.Warn(
				LogConsumerRestartingAfterError,
				attribute.String("group_id", t.groupID),
				attribute.String("topic", t.topic),
			)
			select {
			case <-ctx.Done():
				return
			case <-time.After(consumeRestartBackoff):
			}
		}
	}()
}

func (t *ConsumerDQL[T]) Close() error {
	return t.producerSamara.Close()
}

func NewTopicConsumerGroupDlq[M BaseDqlConsumerModel](config *Config, log logger.Logger, saveFunc SaveFunction, domainFunc ExecDomainFunc[M], options ...ConsumerOption) (kafka.Consumer, error) {
	if config == nil {
		return nil, ErrConfigIsNil
	}
	if domainFunc == nil {
		return nil, ErrExecDomainFuncIsNil
	}
	if config.Topic == "" {
		return nil, ErrTopicRequired
	}
	if config.GroupSettings.GroupID == "" {
		return nil, ErrGroupIDRequired
	}
	if len(config.Brokers) == 0 {
		return nil, ErrBrokersRequired
	}

	baseLogger := log
	if baseLogger == nil {
		baseLogger = logger.DefaultLogger()
	}
	if baseLogger == nil {
		return nil, ErrLoggerIsNil
	}

	cfg := *config

	consumerOptionsState := consumerOptions{}
	for _, option := range options {
		option(&consumerOptionsState)
	}
	if cfg.ConsumeSettings.DlqSave && saveFunc == nil && consumerOptionsState.saveBatch == nil {
		return nil, ErrDLQSaverRequired
	}
	metrics := resolveConsumerMetrics(consumerOptionsState)

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_3_0_0
	if val, ok := offsetMap[cfg.GroupSettings.OffsetInitial]; ok {
		saramaConfig.Consumer.Offsets.Initial = val
	} else {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
		baseLogger.Warn(
			"Unknown offset_initial value, fallback to newest",
			attribute.String("offset_initial", cfg.GroupSettings.OffsetInitial),
			attribute.String("fallback", "new"),
		)
	}
	if val, ok := rebalanceStrategyMap[cfg.GroupSettings.RebalancedGroupStrategy]; ok {
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{val}
	} else {
		saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
		baseLogger.Warn(
			"Unknown rebalance strategy, fallback to round-robin",
			attribute.String("rebalance_strategy", cfg.GroupSettings.RebalancedGroupStrategy),
			attribute.String("fallback", "round-robin"),
		)
	}
	saramaConfig.Consumer.Return.Errors = cfg.GroupSettings.ReturnErrors
	if val, ok := isolationLevelMap[cfg.GroupSettings.IsolationLevel]; ok {
		saramaConfig.Consumer.IsolationLevel = val
	} else {
		saramaConfig.Consumer.IsolationLevel = sarama.ReadCommitted
		baseLogger.Warn(
			"Unknown isolation level, fallback to read_committed",
			attribute.String("isolation_level", cfg.GroupSettings.IsolationLevel),
			attribute.String("fallback", "committed"),
		)
	}
	groupInstanceID := cfg.GroupSettings.GroupInstanceId
	if groupInstanceID == "" {
		groupInstanceID = uuid.New().String()
	}
	saramaConfig.Consumer.Group.InstanceId = groupInstanceID

	if cfg.Network.Sasl.Enable {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = cfg.Network.Sasl.Username
		saramaConfig.Net.SASL.Password = cfg.Network.Sasl.Password

		// Выбор механизма SASL
		switch cfg.Network.Sasl.Mechanism {
		case "SCRAM-SHA-512":
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		case "SCRAM-SHA-256":
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case "PLAIN":
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		default:
			saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
			baseLogger.Warn(
				"Unknown SASL mechanism, fallback to SCRAM-SHA-256",
				attribute.String("sasl_mechanism", cfg.Network.Sasl.Mechanism),
				attribute.String("fallback", "SCRAM-SHA-256"),
			)
		}
	}
	if cfg.Network.TLS.Enabled {
		saramaConfig.Net.TLS.Enable = true
		cert, err := tls.X509KeyPair([]byte(cfg.Network.TLS.ClientCert), []byte(cfg.Network.TLS.ClientKey))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrLoadTLSClientKeyPair, err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(cfg.Network.TLS.RootCert)) {
			return nil, ErrAppendTLSRootCert
		}
		saramaConfig.Net.TLS.Config = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		}
	}

	consumerGroup, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupSettings.GroupID, saramaConfig)
	if err != nil {
		baseLogger.Error(
			LogConsumerCreateGroupFailed,
			attribute.String("group_id", cfg.GroupSettings.GroupID),
			attribute.String("topic", cfg.Topic),
			attribute.String("error", err.Error()),
		)
		return nil, err
	}

	consumerLogger := baseLogger.With(
		attribute.String("group_id", cfg.GroupSettings.GroupID),
		attribute.String("group_instance_id", groupInstanceID),
		attribute.String("topic", cfg.Topic),
	)
	consumerLogger.Info(
		LogConsumerConfigured,
		attribute.String("offset_initial", cfg.GroupSettings.OffsetInitial),
		attribute.String("rebalance_strategy", cfg.GroupSettings.RebalancedGroupStrategy),
		attribute.String("isolation_level", cfg.GroupSettings.IsolationLevel),
		attribute.Int("brokers_count", len(cfg.Brokers)),
	)

	return &ConsumerDQL[M]{
		groupID:        cfg.GroupSettings.GroupID,
		logger:         consumerLogger,
		metrics:        metrics,
		producerSamara: consumerGroup,
		stopped:        make(chan struct{}),
		topic:          cfg.Topic,
		base:           newBaseDqlConsumer[M](consumerLogger, metrics, &cfg, domainFunc, saveFunc, consumerOptionsState.saveBatch),
	}, nil
}

type messageProcessingResult struct {
	message       *sarama.ConsumerMessage
	status        string
	startedAt     time.Time
	processingErr error
	dlqMessage    *DLQMessage
	markReason    string
	logMessage    string
}

func newMessageObject[T BaseDqlConsumerModel]() (T, error) {
	var zero T
	objectType := reflect.TypeOf(any(zero))
	if objectType == nil {
		return zero, ErrMessageObjectTypeNil
	}
	if objectType.Kind() != reflect.Ptr {
		return zero, fmt.Errorf("%w: type=%s", ErrMessageObjectTypePointer, objectType.String())
	}

	object, ok := reflect.New(objectType.Elem()).Interface().(T)
	if !ok {
		return zero, fmt.Errorf("%w: type=%s", ErrCreateMessageObject, objectType.String())
	}

	return object, nil
}
