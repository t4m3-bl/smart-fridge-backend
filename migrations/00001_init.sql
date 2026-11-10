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

CREATE TABLE settings (
    user_id           uuid    PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    has_notifications boolean NOT NULL DEFAULT true,
    notification_time time    NOT NULL DEFAULT '09:00',
    timezone          text    NOT NULL DEFAULT 'Europe/Moscow',
    version           integer NOT NULL DEFAULT 1 CHECK (version > 0)
);

CREATE TABLE indicators (
    id       uuid       PRIMARY KEY,
    user_id  uuid       NOT NULL REFERENCES settings (user_id) ON DELETE CASCADE,
    color    varchar(7) NOT NULL CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
    max_days integer    CHECK (max_days >= 0),

    CONSTRAINT indicators_user_days_uq  UNIQUE NULLS NOT DISTINCT (user_id, max_days),
    CONSTRAINT indicators_user_color_uq UNIQUE (user_id, color) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE categories (
    user_id uuid    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    id      uuid    NOT NULL,
    name    text    NOT NULL CHECK (btrim(name) <> '' AND char_length(name) <= 100),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),

    PRIMARY KEY (user_id, id),
    CONSTRAINT categories_user_name_uq UNIQUE (user_id, name)
);

CREATE TABLE products (
    user_id                  uuid    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    id                       uuid    NOT NULL,
    category_id              uuid    NOT NULL,
    name                     text    NOT NULL CHECK (btrim(name) <> '' AND char_length(name) <= 200),
    description              text    CHECK (char_length(description) <= 2000),
    quantity                 text    CHECK (char_length(quantity) <= 100),
    image_link               text    CHECK (char_length(image_link) <= 2048),
    manufactured_at          date,
    expires_at               date    NOT NULL,
    opened_at                date,
    shelf_life_after_opening integer CHECK (shelf_life_after_opening BETWEEN 1 AND 3650),
    version                  integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now(),
    effective_expires_at     date GENERATED ALWAYS AS (
        CASE
          WHEN opened_at IS NOT NULL AND shelf_life_after_opening IS NOT NULL
              THEN LEAST(expires_at, opened_at + shelf_life_after_opening)
          ELSE expires_at
        END
    ) STORED,

    PRIMARY KEY (user_id, id),
    CONSTRAINT products_category_fk FOREIGN KEY (user_id, category_id) REFERENCES categories (user_id, id),
    CHECK (manufactured_at IS NULL OR manufactured_at <= expires_at),
    CHECK (opened_at IS NULL OR manufactured_at IS NULL OR opened_at >= manufactured_at)
);
CREATE INDEX products_user_expires_idx ON products (user_id, effective_expires_at);
CREATE INDEX products_user_category_idx ON products (user_id, category_id);

-- +goose Down
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS indicators CASCADE;
DROP TABLE IF EXISTS settings CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS identities CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP EXTENSION IF EXISTS citext;
