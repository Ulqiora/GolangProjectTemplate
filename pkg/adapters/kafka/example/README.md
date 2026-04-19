# Kafka Producer & Consumer Metrics Example

Мини-проект показывает, как выглядят метрики продьюсера и консьюмера в Grafana.

## Что поднимается

- Kafka
- Kafka Exporter (cluster/topic/group метрики и consumer lag)
- Demo transactional producer (генерирует события и экспортирует метрики)
- Demo consumer group (читает те же события и экспортирует метрики)
- Prometheus
- Grafana (с автоподключенным datasource и готовыми dashboards)

## Запуск

```bash
cd /Users/andreydamdinov/GolandProjects/GolangProjectTemplate
docker compose -f pkg/adapters/kafka/example/docker-compose.yml up --build
```

## Куда смотреть

- Grafana: http://localhost:3000
  - login: `admin`
  - password: `admin`
  - Dashboards: `Kafka Producer Overview`, `Kafka Consumer Overview`, `Kafka Cluster Overview`
- Prometheus: http://localhost:9090
- Kafka Exporter metrics: http://localhost:9308/metrics
- Метрики demo producer: http://localhost:2112/metrics
- Метрики demo consumer: http://localhost:2113/metrics

## Что увидишь на dashboards

- Producer: throughput, error ratio, latency p95, payload p95, transactional ops, async events
- Consumer: receive rate, processed status rate, processing latency p95, DLQ rate, active sessions, processed total
- Kafka Cluster: consumer lag, top lagging partitions, topic append rate by offsets, log depth, brokers, partitions, under-replicated partitions, groups count

## Нагрузка в этом example

По умолчанию в `docker-compose.yml` выставлен агрессивный профиль:

- Producer: `1000` сообщений каждые `50ms` (теоретически до `~20k msg/s`)
- Payload: `4KB..16KB` на сообщение
- Consumer batch: `2000`, timeout `100ms`

Если нужно еще выше/ниже, меняй env в сервисах `producer-demo` и `consumer-demo`:

- `PRODUCER_BATCH_SIZE`, `PRODUCER_SEND_INTERVAL`, `PRODUCER_PAYLOAD_MIN_BYTES`, `PRODUCER_PAYLOAD_MAX_BYTES`
- `CONSUMER_BATCH_SIZE`, `CONSUMER_BATCH_TIMEOUT`, `CONSUMER_DLQ_SAVE`, `CONSUMER_DLQ_RETRIES`, `CONSUMER_DLQ_TIMEOUT`

## Остановка

```bash
docker compose -f pkg/adapters/kafka/example/docker-compose.yml down -v
```
