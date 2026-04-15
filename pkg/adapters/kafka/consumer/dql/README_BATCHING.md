# Batching в DQL Consumer

## Изменения

### 1. Конфигурация

Добавлены поля в `ConsumeProcessConfig`:

```go
type ConsumeProcessConfig struct {
    // ... существующие поля ...
    
    // Batch settings
    BatchEnabled bool          `yaml:"batch_enabled"`
    BatchSize    int           `yaml:"batch_size"`
    BatchTimeout time.Duration `yaml:"batch_timeout"`
}
```

### 2. Режимы работы

#### Single Mode (поштучная обработка)
```yaml
batch_enabled: false
```

Сообщения обрабатываются по одному с коммитом после каждого.

#### Batch Mode
```yaml
batch_enabled: true
batch_size: 100
batch_timeout: 100ms
```

Сообщения накапливаются и обрабатываются батчами по:
- Достижению `batch_size`
- Истечению `batch_timeout`

### 3. Производительность

Ожидаемые результаты (1000 сообщений, 1ms обработка):

| Режим | Batch Size | Throughput | Улучшение |
|-------|-----------|------------|-----------|
| Single | - | ~800 msg/sec | baseline |
| Batch | 10 | ~1500 msg/sec | 1.9x |
| Batch | 50 | ~2800 msg/sec | 3.5x |
| Batch | 100 | ~3500 msg/sec | 4.4x |
| Batch | 200 | ~3900 msg/sec | 4.9x |

### 4. Пример конфигурации

```yaml
kafka_consumer:
  topic: "user-events"
  brokers:
    - "localhost:9092"
  
  group_settings:
    group_id: "user-events-consumer"
    offset_initial: "old"
  
  # Включить батчинг
  batch_enabled: true
  batch_size: 100
  batch_timeout: 100ms
  
  # DLQ
  save_to_dlq: true
  dlq_retries: 3
```

### 5. Запуск тестов

```bash
# Интеграционные тесты
go test -v -tags=integration -timeout=10m ./pkg/adapters/kafka/consumer/dql/...

# Конкретный тест
go test -v -tags=integration -run=TestConsumerPerformance ./pkg/adapters/kafka/consumer/dql/...
```

### 6. Рекомендации

| Сценарий | Batch Size | Timeout | 
|----------|-----------|---------|
| Real-time | 1-10 | 10-50ms |
| Standard | 50-100 | 100ms |
| High-load | 100-200 | 200-500ms |
| Analytics | 200-500 | 500ms-1s |
