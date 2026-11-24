-- +goose Up

-- push_subscriptions
CREATE TABLE push_subscriptions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id	uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    endpoint text NOT NULL UNIQUE,
    p256dh text NOT NULL,
    auth text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz
);

CREATE INDEX push_subscriptions_active_user_idx
    ON push_subscriptions (user_id)
    WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS push_subscriptions;