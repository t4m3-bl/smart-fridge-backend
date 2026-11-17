-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id            uuid        PRIMARY KEY,
    email         citext      NOT NULL UNIQUE,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX users_email_idx ON users (email);

CREATE TABLE identities (
    id               uuid        PRIMARY KEY,
    user_id          uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider         varchar(50) NOT NULL,
    provider_user_id text,
    password_hash    text,
    created_at       timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT oauth_identities_provider_user_key UNIQUE (provider, provider_user_id),
    CONSTRAINT oauth_identities_user_provider_key UNIQUE (user_id, provider),
    CONSTRAINT identities_native_password_chk CHECK (
        (provider IN ('yandex', 'native')) AND
        ((provider = 'native' AND password_hash IS NOT NULL AND provider_user_id = '') OR
        (provider <> 'native' AND password_hash IS NULL AND provider_user_id <> ''))
    )
);
CREATE INDEX identities_provider_uid_idx ON identities (provider, provider_user_id);

CREATE TABLE sessions (
    id                 uuid        PRIMARY KEY,
    user_id            uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    refresh_token_hash text        NOT NULL UNIQUE,
    created_at         timestamptz NOT NULL DEFAULT now(),
    expires_at         timestamptz NOT NULL,
    revoked_at         timestamptz,
    CHECK (expires_at > created_at)
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);

-- +goose Down
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS identities CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP EXTENSION IF EXISTS citext;
