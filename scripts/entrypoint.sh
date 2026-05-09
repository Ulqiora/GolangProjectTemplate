#!/bin/bash
set -e

wait_for_db() {
  until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER"; do
    echo "Waiting for database..."
    sleep 2
  done
}

run_migrations() {
  echo "Running migrations..."
  goose -dir="${MIGRATIONS_DIR:-migrations/postgres}" postgres \
    "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE:-disable}" \
    up
}

wait_for_db
run_migrations
echo "Migrations completed."
