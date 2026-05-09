-- +goose Up
-- +goose NO TRANSACTION
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS message_id_unique_idx on dql_message (id);
-- +goose NO TRANSACTION

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX IF EXISTS message_id_unique_idx;
-- +goose NO TRANSACTION
