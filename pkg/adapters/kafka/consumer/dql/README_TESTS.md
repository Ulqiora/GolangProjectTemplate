# Performance Tests for DQL Consumer

Тесты для измерения производительности Kafka консьюмера с поддержкой батчинга.

## Требования

- Docker (для testcontainers)
- Go 1.25+
- Минимум 2GB свободной памяти

## Запуск тестов

### Все тесты (интеграционные)
```bash
go test -v -tags=integration -timeout=10m ./pkg/adapters/kafka/consumer/dql/...
```

### Только тесты производительности
```bash
go test -v -tags=integration -run=TestConsumerPerformance -timeout=10m ./pkg/adapters/kafka/consumer/dql/...
```

### Тест максимальной пропускной способности
```bash
go test -v -tags=integration -run=TestConsumerThroughput -timeout=10m ./pkg/adapters/kafka/consumer/dql/...
```

### Короткий режим (без performance тестов)
```bash
go test -short ./pkg/adapters/kafka/consumer/dql/...
```

## Описание тестов

### 1. TestConsumerPerformance_SingleVsBatch
Сравнивает производительность разных режимов работы:
- **Single Mode** - поштучная обработка сообщений
- **Batch 10** - батчи по 10 сообщений
- **Batch 50** - батчи по 50 сообщений
- **Batch 100** - батчи по 100 сообщений

**Параметры:**
- Количество сообщений: 1000
- Задержка обработки: 1ms на сообщение
- Timeout батча: 100ms

### 2. TestConsumerThroughput_MaxThroughput
Измеряет максимальную пропускную способность для разных конфигураций:
- NoBatch (поштучная обработка)
- Batch 10, 50, 100, 200

**Параметры:**
- Количество сообщений: 5000
- Без искусственной задержки

### 3. TestConsumerBatchTimeout
Проверяет работу таймаута батча:
- Отправляется 5 сообщений
- Размер батча: 100 (больше чем сообщений)
- Таймаут: 500ms
- Ожидается, что обработка займёт ~500ms

## Ожидаемые результаты

### Пример вывода TestConsumerPerformance_SingleVsBatch:
```
=== RUN   TestConsumerPerformance_SingleVsBatch
=== RUN   TestConsumerPerformance_SingleVsBatch/Single_Mode_1000
    ✓ Completed: 1000 messages in 1.234s
      Throughput: 810.37 msg/sec
      Batch enabled: false, Batch size: 0
=== RUN   TestConsumerPerformance_SingleVsBatch/Batch_10_Mode_1000
    ✓ Completed: 1000 messages in 0.567s
      Throughput: 1763.67 msg/sec
      Batch enabled: true, Batch size: 10
=== RUN   TestConsumerPerformance_SingleVsBatch/Batch_50_Mode_1000
    ✓ Completed: 1000 messages in 0.345s
      Throughput: 2898.55 msg/sec
      Batch enabled: true, Batch size: 50
=== RUN   TestConsumerPerformance_SingleVsBatch/Batch_100_Mode_1000
    ✓ Completed: 1000 messages in 0.289s
      Throughput: 3460.21 msg/sec
      Batch enabled: true, Batch size: 100
```

### Пример вывода TestConsumerThroughput_MaxThroughput:
```
========== PERFORMANCE COMPARISON ==========
NoBatch        :   850.45 msg/sec (5.876s)
Batch_10       :  1523.67 msg/sec (3.282s)
Batch_50       :  2845.12 msg/sec (1.758s)
Batch_100      :  3567.89 msg/sec (1.401s)
Batch_200      :  3890.23 msg/sec (1.285s)
============================================
```

## Конфигурация батчинга

Для включения батчинга в production используйте следующие настройки в `config.yaml`:

```yaml
group_settings:
  group_id: "outbox-consumer"
  offset_initial: "old"
  return_errors: true
  isocation_level: "commited"

# Batch settings
batch_enabled: true      # Включить батчинг
batch_size: 100          # Размер батча (оптимально 50-200)
batch_timeout: 100ms     # Таймаут накопления батча

# DLQ settings
save_to_dlq: true
dlq_retries: 3
```

## Рекомендации по выбору размера батча

| Размер батча | Плюсы | Минусы | Когда использовать |
|-------------|-------|--------|-------------------|
| 1-10 | Низкая задержка | Низкая пропускная способность | Real-time обработка, мало сообщений |
| 10-50 | Баланс задержки и throughput | - | Стандартный сценарий |
| 50-200 | Высокая пропускная способность | Больше задержка | High-load, пакетная обработка |
| 200+ | Максимальный throughput | Высокая задержка, больше память | Аналитика, не критично к задержкам |

## Troubleshooting

### Docker не запускается
```bash
# Проверьте, что Docker запущен
docker ps

# Проверьте права доступа
sudo usermod -aG docker $USER
```

### Testcontainers timeout
Увеличьте timeout:
```bash
export TESTCONTAINERS_RYUK_DISABLED=true
go test -v -tags=integration -timeout=15m ./...
```

### Недостаточно памяти
Уменьшите количество сообщений в тестах или увеличьте память Docker:
```yaml
# deployment/docker-compose.kafka.yaml
services:
  kafka:
    mem_limit: 1g
```

## Метрики

Для мониторинга производительности в production рекомендуется использовать:

1. **Prometheus метрики**:
   - `kafka_consumer_messages_total` - всего обработано сообщений
   - `kafka_consumer_batch_size` - размер батча
   - `kafka_consumer_processing_duration_seconds` - время обработки

2. **Grafana дашборды**:
   - Throughput (msg/sec)
   - Latency (p50, p95, p99)
   - Batch fill rate

## Дальнейшие улучшения

- [ ] Добавить тесты с конкурирующими консьюмерами
- [ ] Тесты на отказоустойчивость (rebalance)
- [ ] Бенчмарки с разной величиной сообщений
- [ ] Интеграция с Jaeger для трейсинга
