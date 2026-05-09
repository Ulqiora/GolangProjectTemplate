-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth_users
(
    id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    login      text        NOT NULL,
    email      text        NOT NULL,
    first_name text        NOT NULL,
    last_name  text        NOT NULL,
    provider   text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT auth_users_login_unique UNIQUE (login),
    CONSTRAINT auth_users_email_unique UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS auth_password_credentials
(
    user_id       uuid PRIMARY KEY,
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT auth_password_credentials_user_fk
        FOREIGN KEY (user_id) REFERENCES auth_users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS auth_external_identities
(
    id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     uuid        NOT NULL,
    provider   text        NOT NULL,
    subject    text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT auth_external_identities_provider_subject_unique UNIQUE (provider, subject),
    CONSTRAINT auth_external_identities_user_fk
        FOREIGN KEY (user_id) REFERENCES auth_users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS auth_outbox_messages
(
    id              uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    topic           text        NOT NULL,
    message_key     text        NOT NULL,
    payload         bytea       NOT NULL,
    status          text        NOT NULL,
    attempt_count   integer     NOT NULL DEFAULT 0,
    last_error      text        NOT NULL DEFAULT '',
    published_at    timestamptz NULL,
    next_attempt_at timestamptz NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS auth_outbox_messages_status_idx
    ON auth_outbox_messages (status, created_at);

CREATE INDEX IF NOT EXISTS auth_outbox_messages_retry_idx
    ON auth_outbox_messages (status, next_attempt_at, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS auth_outbox_messages;
DROP TABLE IF EXISTS auth_external_identities;
DROP TABLE IF EXISTS auth_password_credentials;
DROP TABLE IF EXISTS auth_users;
-- +goose StatementEnd
