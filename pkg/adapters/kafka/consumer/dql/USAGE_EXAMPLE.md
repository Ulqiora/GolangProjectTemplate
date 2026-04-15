# Пример использования DQL Consumer с батчингом

## Базовый пример

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    dql "GolangTemplateProject/pkg/adapters/kafka/consumer/dql"
    "GolangTemplateProject/pkg/logger"
)

// Ваша модель данных
type UserEvent struct {
    UserID    string    `json:"user_id"`
    Event     string    `json:"event"`
    Timestamp time.Time `json:"timestamp"`
}

// Реализация интерфейсов BaseModel и MessageObject
func (u *UserEvent) Params() map[string]interface{} {
    return map[string]interface{}{
        "user_id": u.UserID,
        "event": u.Event,
        "timestamp": u.Timestamp,
    }
}

func (u *UserEvent) Fields() []string {
    return []string{"user_id", "event", "timestamp"}
}

func (u *UserEvent) PrimaryKey() (string, any) {
    return "user_id", u.UserID
}

func (u *UserEvent) Marshal() ([]byte, error) {
    return json.Marshal(u)
}

func (u *UserEvent) Unmarshal(data []byte) error {
    return json.Unmarshal(data, u)
}

func main() {
    ctx := context.Background()
    
    // Конфигурация
    config := &dql.Config{
        Topic:   "user-events",
        Brokers: dql.Brokers{"localhost:9092", "localhost:9093"},
        GroupSettings: dql.GroupSettings{
            GroupID:        "user-events-consumer",
            OffsetInitial:  "old",
            ReturnErrors:   true,
            IsolationLevel: "commited",
        },
        ConsumeProcessConfig: dql.ConsumeProcessConfig{
            // Включаем батчинг
            BatchEnabled: true,
            BatchSize:    100,
            BatchTimeout: 100 * time.Millisecond,
            
            // DLQ настройки
            DlqSave:    true,
            DlqRetries: 3,
        },
    }
    
    // Функция обработки сообщения
    execFunc := func(ctx context.Context, event *UserEvent, values *dql.MapValues) error {
        fmt.Printf("Processing event: %+v\n", event)
        // Ваша бизнес-логика здесь
        return nil
    }
    
    // Функция сохранения в DLQ
    saveFunc := func(ctx context.Context, dlqMsg *dql.DLQMessage) error {
        fmt.Printf("Saving to DLQ: %s\n", dlqMsg.ID)
        // Сохранение в БД
        return nil
    }
    
    // Создаём консьюмер
    consumer, err := dql.NewTopicConsumerGroupDlq[*UserEvent](
        config,
        logger.DefaultLogger(),
        saveFunc,
        execFunc,
    )
    if err != nil {
        panic(err)
    }
    
    // Запускаем
    consumer.Run(ctx)
    
    // Ждём завершения
    <-ctx.Done()
}
```

## Конфигурация из YAML

```yaml
# config.yaml
kafka_consumer:
  topic: "user-events"
  brokers:
    - "localhost:9092"
    - "localhost:9093"
  
  group_settings:
    group_id: "user-events-consumer"
    rebalance_strategy: "round-robin"
    offset_initial: "old"
    return_errors: true
    isocation_level: "commited"
    group_instance_id: "" # оставьте пустым для автоматической генерации
  
  # Настройки батчинга
  batch_enabled: true
  batch_size: 100
  batch_timeout: 100ms
  
  # DLQ настройки
  save_to_dlq: true
  dlq_timeout: 5s
  dlq_retries: 3
  
  # Таймаут чтения сообщения
  timeout_reading_message: 30s
  message_processing_retries: 3
  
  # TLS (опционально)
  network:
    tls:
      enabled: true
      client-cert: |
        -----BEGIN CERTIFICATE-----
        ...
        -----END CERTIFICATE-----
      client-key: |
        -----BEGIN PRIVATE KEY-----
        ...
        -----END PRIVATE KEY-----
      root-cert: |
        -----BEGIN CERTIFICATE-----
        ...
        -----END CERTIFICATE-----
    
    # SASL (опционально)
    sasl:
      enable: true
      username: "user"
      password: "password"
      mechanism: "SCRAM-SHA-256" # PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
```

Загрузка конфигурации:

```go
import "gopkg.in/yaml.v2"

func loadConfig(path string) (*dql.Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var config dql.Config
    err = yaml.Unmarshal(data, &config)
    if err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

## Режимы работы

### 1. Поштучная обработка (без батчинга)

```go
config.ConsumeProcessConfig = dql.ConsumeProcessConfig{
    BatchEnabled: false,
    DlqSave:      true,
}
```

**Когда использовать:**
- Real-time обработка критична
- Мало сообщений (< 100/sec)
- Простая бизнес-логика

### 2. Батчинг по размеру

```go
config.ConsumeProcessConfig = dql.ConsumeProcessConfig{
    BatchEnabled: true,
    BatchSize:    100,
    BatchTimeout: 1 * time.Second, // фоллбэк
    DlqSave:      true,
}
```

**Когда использовать:**
- Высокий throughput (> 1000 msg/sec)
- Пакетная обработка в БД
- Аналитика

### 3. Батчинг по таймауту

```go
config.ConsumeProcessConfig = dql.ConsumeProcessConfig{
    BatchEnabled: true,
    BatchSize:    1000, // большой размер
    BatchTimeout: 100 * time.Millisecond, // низкая задержка
    DlqSave:      true,
}
```

**Когда использовать:**
- Баланс между задержкой и throughput
- Неравномерный поток сообщений

## Обработка ошибок

```go
execFunc := func(ctx context.Context, event *UserEvent, values *dql.MapValues) error {
    // Валидация
    if event.UserID == "" {
        return fmt.Errorf("invalid user_id")
    }
    
    // Бизнес-логика
    err := processEvent(event)
    if err != nil {
        // Ошибка будет сохранена в DLQ (если DlqSave=true)
        return err
    }
    
    return nil
}
```

## Метрики и мониторинг

Добавьте логирование для метрик:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    messagesProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kafka_consumer_messages_total",
            Help: "Total processed messages",
        },
        []string{"topic", "status"},
    )
    
    batchSize = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "kafka_consumer_batch_size",
            Help:    "Batch size distribution",
            Buckets: prometheus.ExponentialBuckets(1, 2, 10),
        },
    )
    
    processingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "kafka_consumer_processing_duration_seconds",
            Help:    "Message processing duration",
            Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
        },
        []string{"topic"},
    )
)

func init() {
    prometheus.MustRegister(messagesProcessed, batchSize, processingDuration)
}
```

## Graceful Shutdown

```go
import (
    "context"
    "os"
    "os/signal"
    "syscall"
)

func runConsumerWithShutdown(consumer kafka.Consumer) {
    ctx, cancel := context.WithCancel(context.Background())
    
    // Запуск консьюмера
    consumer.Run(ctx)
    
    // Ожидание сигнала
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    <-sigChan
    fmt.Println("Shutting down...")
    
    // Отмена контекста
    cancel()
    
    // Ожидание завершения
    select {
    case <-consumer.WaitStoppedSession():
        fmt.Println("Consumer stopped gracefully")
    case <-time.After(30 * time.Second):
        fmt.Println("Consumer shutdown timeout")
    }
}
```

## Тестирование

```go
//go:build integration
// +build integration

func TestMyConsumer(t *testing.T) {
    ctx := context.Background()
    
    // Поднять Kafka через testcontainers
    kafkaContainer, _ := kafka.RunContainer(ctx,
        kafka.WithClusterID("test-cluster"),
        testcontainers.WithImage("confluentinc/cp-kafka:7.5.0"),
    )
    defer kafkaContainer.Terminate(ctx)
    
    broker, _ := kafkaContainer.Broker(ctx)
    
    // Создать консьюмер
    config := &dql.Config{
        Topic:   "test-topic",
        Brokers: dql.Brokers{broker},
        ConsumeProcessConfig: dql.ConsumeProcessConfig{
            BatchEnabled: true,
            BatchSize:    10,
        },
    }
    
    // ... тестирование
}
```

## Рекомендации

### Оптимальные настройки для разных сценариев

| Сценарий | Batch Size | Batch Timeout | Throughput | Latency |
|----------|-----------|---------------|------------|---------|
| Real-time | 1-10 | 10-50ms | Низкий | < 50ms |
| Standard | 50-100 | 100ms | Средний | 100-200ms |
| High-load | 100-200 | 200-500ms | Высокий | 200-500ms |
| Analytics | 200-500 | 500ms-1s | Максимальный | > 500ms |

### Чеклист перед production

- [ ] Настроить DLQ для ошибочных сообщений
- [ ] Включить TLS/SASL для безопасности
- [ ] Настроить алерты на ошибки
- [ ] Добавить метрики в Prometheus
- [ ] Протестировать rebalance
- [ ] Проверить graceful shutdown
- [ ] Настроить логирование
