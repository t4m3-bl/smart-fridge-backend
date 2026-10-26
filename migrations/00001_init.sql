-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id            uuid        PRIMARY KEY,
    email         citext      NOT NULL UNIQUE,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX users_email_idx ON users (email);

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

CREATE TABLE settings (
    user_id           uuid    PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    has_notifications boolean NOT NULL DEFAULT true,
    notification_time time    NOT NULL DEFAULT '09:00',
    timezone          text    NOT NULL DEFAULT 'Europe/Moscow'
);

CREATE TABLE indicators (
    id       uuid       PRIMARY KEY,
    user_id  uuid       NOT NULL REFERENCES settings (user_id) ON DELETE CASCADE,
    color    varchar(7) NOT NULL CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
    max_days integer    CHECK (max_days >= 0),
    CONSTRAINT indicators_user_days_uq  UNIQUE NULLS NOT DISTINCT (user_id, max_days),
    CONSTRAINT indicators_user_color_uq UNIQUE (user_id, color) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE products (
    id                       uuid    PRIMARY KEY,
    user_id                  uuid    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name                     text    NOT NULL CHECK (btrim(name) <> ''),
    category                 text,
    description              text,
    quantity                 text,
    image_link               text,
    manufactured_at          date,
    expires_at               date    NOT NULL,
    opened_at                date,
    shelf_life_after_opening integer CHECK (shelf_life_after_opening > 0),
    effective_expires_at     date GENERATED ALWAYS AS (
        CASE
            WHEN opened_at IS NOT NULL AND shelf_life_after_opening IS NOT NULL
                THEN LEAST(expires_at, opened_at + shelf_life_after_opening)
            ELSE expires_at
        END
    ) STORED,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now(),
    CHECK (manufactured_at IS NULL OR manufactured_at <= expires_at),
    CHECK (opened_at IS NULL OR manufactured_at IS NULL OR opened_at >= manufactured_at)
);

CREATE INDEX products_user_expires_idx ON products (user_id, effective_expires_at);

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

-- +goose Down
DROP TABLE products;
DROP TABLE indicators;
DROP TABLE settings;
DROP TABLE sessions;
DROP TABLE users;
DROP TABLE identities;
