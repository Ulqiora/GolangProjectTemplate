#!/bin/bash
set -e

# Функция ожидания БД
wait_for_db() {
  until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER"; do
    echo "Waiting for database..."
    sleep 2
  done
}

# Выполнение миграций
run_migrations() {
  echo "Running migrations..."
  goose -dir=migrations postgres "postgres://postgres:postgres@postgresql-db:5432/postgres?sslmode=disable"  up
}

# Основной цикл
wait_for_db
run_migrations

# Бесконечное ожидание (чтобы контейнер не завершался)
echo "Migrations completed. Keeping container alive..."
while true; do
  sleep 86400  # 1 день (можно уменьшить)
done