-- +goose Up

-- categories
CREATE TABLE categories (
    id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL CHECK (btrim(name) <> '' AND char_length(name) <= 100)
);

INSERT INTO categories (name)
VALUES
    ('Dairy'),
    ('Meat and fish'),
    ('Vegetables and fruits'),
    ('Ready-to-eat meals'),
    ('Beverages'),
    ('Other');

-- products
CREATE TABLE products (
    id                       uuid        NOT NULL,
    user_id                  uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    category_id              BIGINT      NOT NULL REFERENCES categories (id),
    name                     text        NOT NULL CHECK (btrim(name) <> '' AND char_length(name) <= 200),
    description              text        CHECK (char_length(description) <= 2000),
    quantity                 text        CHECK (char_length(quantity) <= 100),
    image_key                text,
    manufactured_at          date,
    expires_at               date        NOT NULL,
    opened_at                date,
    shelf_life_after_opening integer     CHECK (shelf_life_after_opening BETWEEN 1 AND 3650),
    version                  integer     NOT NULL DEFAULT 1 CHECK (version > 0),
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
    CHECK (manufactured_at IS NULL OR manufactured_at <= expires_at),
    CHECK (opened_at IS NULL OR manufactured_at IS NULL OR opened_at >= manufactured_at)
);

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

-- +goose Down
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS settings CASCADE;
DROP TABLE IF EXISTS indicators CASCADE;
