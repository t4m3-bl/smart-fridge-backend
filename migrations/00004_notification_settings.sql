-- +goose Up
ALTER TABLE push_subscriptions ADD COLUMN logout_token_hash text;

-- +goose Down
ALTER TABLE push_subscriptions DROP COLUMN logout_token_hash;
