-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_registration
(
    id         uuid                     DEFAULT uuid_generate_v4(),
    email      TEXT NOT NULL,
    firstname  TEXT NOT NULL,
    lastname   TEXT NOT NULL,
    login      TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT u_login UNIQUE (login),
    CONSTRAINT u_email UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS user_secret
(
    user_id         uuid NOT NULL,
    hashed_password TEXT NOT NULL,
    otp_url         TEXT NOT NULL,
    otp_nonce       TEXT NOT NULL,
    otp_secret      TEXT NOT NULL,
    refresh_token   TEXT NOT NULL,
    access_token    TEXT NOT NULL,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT u_user UNIQUE (user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "user_registration";
DROP TABLE IF EXISTS "user_secret";
-- +goose StatementEnd
