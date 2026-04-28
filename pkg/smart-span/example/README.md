# Smart Span Example

Мини-проект поднимает:

- `smart-span-demo` c реальным `checkout` flow
- `Jaeger UI`

## Запуск

```bash
cd /Users/andreydamdinov/GolandProjects/GolangProjectTemplate
docker compose -f pkg/smart-span/example/docker-compose.yml up --build
```

## Куда смотреть

- demo app: [http://localhost:18080](http://localhost:18080)
- jaeger: [http://localhost:16686](http://localhost:16686)

## Что делать

1. Открой [http://localhost:16686/search](http://localhost:16686/search)
2. Выбери service `smart-span-demo`
3. Открой свежий trace

Или вызови flow вручную:

```bash
curl http://localhost:18080/demo/checkout
curl "http://localhost:18080/demo/checkout?fail=true"
```

Ответ вернет:

- `trace_id`
- `jaeger_trace_url`
- `jaeger_search_url`

Поэтому можно сразу открыть конкретный trace.

## Какой flow увидишь

Root span:

- `checkout.flow`

Child spans:

- `checkout.validate_customer`
- `checkout.reserve_inventory`
- `checkout.charge_payment`
- `checkout.publish_order_event`

Фоновый генератор раз в `5s` отправляет новый flow и чередует:

- успешный checkout
- checkout с ошибкой оплаты

Поэтому traces будут появляться в Jaeger даже без ручных запросов.

## Остановка

```bash
docker compose -f pkg/smart-span/example/docker-compose.yml down -v
```
