# KODA.md — Инструкции для работы с проектом

## Обзор проекта

Это **Go-проект** (GolangTemplateProject), построенный по принципам **Clean Architecture**. Проект представляет собой микросервис для обработки данных с использованием современных технологий.

### Основные технологии

| Технология | Назначение |
|------------|------------|
| Go 1.24.4 | Язык программирования |
| Fiber v2 | HTTP-сервер и маршрутизация |
| gRPC + Protobuf | API и сериализация данных |
| PostgreSQL | База данных |
| Kafka (Sarama) | Очередь сообщений |
| OpenTelemetry + Jaeger | Трейсинг и наблюдаемость |
| Prometheus + Grafana | Метрики и мониторинг |
| ClickHouse | Аналитическая БД |

### Архитектура проекта

```
GolangTemplateProject/
├── cmd/                    # Точки входа приложений
│   ├── outbox/            # Основное приложение (outbox-обработка)
│   └── runner/            # Приложение runner
├── internal/              # Внутренние пакеты (Clean Architecture)
│   ├── app/outbox/        # Приложение outbox
│   ├── domain/            # Доменные модели
│   ├── usecase/           # Бизнес-логика
│   ├── repository/        # Репозитории (работа с БД)
│   ├── ports/             # Порт-интерфейсы
│   ├── adapters/          # Адаптеры
│   │   ├── primary/       # Первичные адаптеры (HTTP, gRPC, k8s)
│   │   └── secondary/     # Вторичные адаптеры (Kafka, БД)
│   └── config/            # Конфигурация
├── pkg/                   # Переиспользуемые пакеты
│   ├── adapters/          # Адаптеры (Kafka, PostgreSQL, gRPC)
│   ├── jwt/               # Работа с JWT
│   ├── logger/            # Логирование
│   ├── otp/               # OTP-генерация
│   ├── prometheus/        # Метрики Prometheus
│   ├── jobs/              # Планировщик задач
│   └── ...                # Другие утилиты
├── api/                   # API определения
│   └── user/user.proto    # Protobuf-определения
├── deployment/            # Docker и деплой-конфигурации
│   ├── docker-compose.*.yaml
│   ├── outbox/Dockerfile
│   └── goose/Dockerfile
├── migrations/            # Миграции БД (goose)
├── scripts/               # Скрипты сборки и генерации
├── config.yaml            # Конфигурация приложения
├── go.mod                 # Зависимости Go
└── Makefile               # Команды сборки
```

---

## Сборка и запуск

### Предварительные требования

- Go 1.24+
- Docker и Docker Compose
- OpenSSL (для генерации TLS-сертификатов)

### Основные команды Makefile

```bash
# Установка всех зависимостей и запуск сервисов
make dependup

# Сборка приложения
make build

# Сборка Docker-образа
make docker-image

# Отладка с Delve
make debug

# Генерация gRPC-кода из proto-файлов
make generate-api

# Создание новой миграции
make goose-create migration-name=<имя>
```

### Запуск локальной инфраструктуры

```bash
# Поднять PostgreSQL, Kafka, Jaeger
make dependup

# Или вручную:
NETWORK_NAME=MYSERVICE docker-compose -f deployment/docker-compose.database.yaml up -d
NETWORK_NAME=MYSERVICE docker-compose -f deployment/docker-compose.kafka.yaml up -d
NETWORK_NAME=MYSERVICE docker-compose -f deployment/docker-compose.tracing.yaml up -d
```

### Конфигурация

Основной файл конфигурации: `config.yaml`

```yaml
server_info:
  name: "service-test"
  grpc_connection:
    host: "localhost"
    port: "10000"
  http_connection:
    host: "localhost"
    port: "10001"

database:
  postgres:
    host: "localhost"
    port: "5435"
    user: "postgres"
    password: "postgres"
    database: "postgres"

kafka:
  brokers: ["localhost:29092"]
  topic: "user"

trace:
  jaeger:
    endpoint: "http://jaeger:14268/api/traces"
```

### Порты сервисов

| Сервис | Порт |
|--------|------|
| Приложение (HTTP) | 10001 |
| PostgreSQL | 5435 |
| Kafka (host) | 9092 |
| Kafka (docker) | 29092 |
| Jaeger UI | 14268 |
| Schema Registry | 8081 |
| Kafka UI | 8080 |
| ClickHouse HTTP | 8123 |
| Prometheus | 9090 |
| Grafana | 3000 |

---

## Структура кода

### Доменные модели (`internal/domain/`)

- `user.go` — Модель пользователя
- `user-secrets.go` — Секреты пользователя
- `registration.go` — Регистрация
- `dlq-message.go` — Сообщения DLQ

### Репозитории (`internal/repository/`)

- `user/` — Репозиторий пользователей

### Use Cases (`internal/usecase/`)

- `outbox/` — Бизнес-логика outbox-обработки
- `authorization/` — Авторизация и валидация паролей

### Адаптеры

**Первичные (входящие):**
- `internal/adapters/primary/user/` — HTTP-обработчики
- `internal/adapters/primary/proto/` — gRPC-определения
- `internal/adapters/primary/prometheus/` — Метрики
- `internal/adapters/primary/k8s/` — Kubernetes-интеграции

**Вторичные (исходящие):**
- `pkg/adapters/kafka/` — Kafka-продюсеры и консьюмеры
- `pkg/adapters/postgres/` — PostgreSQL-соединение

---

## Разработка

### Генерация gRPC-кода

```bash
# Загрузить необходимые protobuf-инструменты
make .load-proto-bins

# Сгенерировать Go-код из proto-файлов
make generate-api
```

Proto-файлы находятся в:
- `internal/adapters/primary/proto/user/user.proto`
- `api/user/user.proto`

Сгенерированный код:
- `internal/adapters/primary/generated/user/`

### Работа с миграциями

```bash
# Создать новую миграцию
make goose-create migration-name=add_new_table sql

# Применить миграции (через Docker)
docker-compose -f deployment/docker-compose.database.yaml up migrate
```

### TLS-сертификаты

```bash
# Генерация CA и сертификатов для mTLS
make gen-mtls-ca

# Генерация TLS-сертификата для сервера
make gen-tls-ca
```

---

## Тестирование

**Текущий статус:** Тесты в проекте отсутствуют (`**/*_test.go` не найдены).

**Рекомендации:**
- Добавить unit-тесты для use cases
- Добавить интеграционные тесты для репозиториев
- Использовать стандартную библиотеку `testing` или `testify`

---

## Стиль кодирования

### Общие принципы

- Использование Clean Architecture с чётким разделением слоёв
- Использование интерфейсов для зависимостей (ports)
- Логирование через `slog` и собственный `pkg/logger`
- Трейсинг через OpenTelemetry
- Graceful shutdown через `pkg/closer`

### Структура обработчика (пример)

```go
func NewApplication(ctx context.Context) (*Application, error) {
    app := new(Application)
    err := app.SetupDependencies(ctx)
    if err != nil {
        return nil, err
    }
    return app, nil
}

func (a *Application) SetupDependencies(ctx context.Context) error {
    // Загрузка конфигурации
    err := config.LoadConfig()
    if err != nil {
        panic(err)
    }
    
    // Инициализация БД
    pool, err := postgres.New(ctx, &config.Get().Database.Postgres)
    // ...
}
```

---

## Развёртывание

### Docker

```bash
# Собрать образ
make docker-image BUILD_EXEC=outbox

# Запустить приложение
NETWORK_NAME=MYSERVICE docker-compose -f deployment/docker-compose.app.yaml up -d
```

### Структура Docker Compose

- `docker-compose.database.yaml` — PostgreSQL + миграции
- `docker-compose.kafka.yaml` — Kafka, Zookeeper, Schema Registry, Kafka UI, ClickHouse
- `docker-compose.tracing.yaml` — Jaeger
- `docker-compose.app.yaml` — Приложение

---

## TODO и известные ограничения

- [ ] Добавить unit-тесты
- [ ] Добавить интеграционные тесты
- [ ] Реализовать полноценные HTTP/gRPC-эндпоинты (в коде есть закомментированные)
- [ ] Настроить CI/CD
- [ ] Добавить линтер и форматтер кода
