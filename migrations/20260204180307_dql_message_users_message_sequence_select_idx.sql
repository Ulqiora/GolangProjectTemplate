-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX CONCURRENTLY IF NOT EXISTS message_sequence_select_idx on dql_message (attempt_number, last_attempt_error);
-- +goose NO TRANSACTION

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX IF EXISTS message_sequence_select_idx;
-- +goose NO TRANSACTION
