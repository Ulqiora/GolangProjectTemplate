# Kafka Producer Metrics Example

Мини-проект показывает, как выглядят метрики продьюсера в Grafana.

## Что поднимается

- Kafka
- Demo transactional producer (генерирует события и экспортирует метрики)
- Prometheus
- Grafana (с автоподключенным datasource и готовым dashboard)

## Запуск

```bash
cd kafka/example
docker compose up --build
```

## Куда смотреть

- Grafana: http://localhost:3000
  - login: `admin`
  - password: `admin`
  - Dashboard: `Kafka Producer Overview` (папка `Kafka`)
- Prometheus: http://localhost:9090
- Метрики demo producer: http://localhost:2112/metrics

## Что увидишь на дашборде

- Send Throughput (msg ops/s)
- Error Ratio
- Operation Latency p95
- Payload Size p95
- Transactional Operations
- Async Producer Events

## Остановка

```bash
docker compose down -v
```
