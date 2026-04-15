-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS dql_message
(
    id                             uuid        NOT NULL DEFAULT uuid_generate_v4(),
    -- message position in kafka cluster
    topic                          varchar     NOT NULL,
    partition                      INT         NOT NULL,
    message_offset                 BIGINT      NOT NULL,
    object_index                   BIGINT      NOT NULL default 0,
    -- internal object info
    payload                        jsonb       NOT NULL default '',
    -- error
    status                         varchar     not null default 'pending',
    attempt_number                 INT         NOT NULL DEFAULT 1,
    last_attempt_error             VARCHAR     NOT NULL default '',
    last_attempt_error_description VARCHAR     NOT NULL default '',
    last_attempt_time              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    deleted                        bool        NOT NULL DEFAULT false,

    -- system message parameters
    created_at                     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
COMMENT ON COLUMN dql_message_users.object_index IS 'index in message of list objects';
COMMENT ON COLUMN dql_message_users.payload IS 'content of one object in message';
COMMENT ON COLUMN dql_message_users.last_attempt_time IS 'time of last attempt object processing';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS dql_message_users;
-- +goose StatementEnd
