-- +goose Up
-- +goose NO TRANSACTION
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS message_position_unique_idx on dql_message (topic, partition, message_offset, object_index);
-- +goose NO TRANSACTION

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX IF EXISTS message_position_unique_idx;
-- +goose NO TRANSACTION
