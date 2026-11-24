-- name: UpsertPushSubscription :one
INSERT INTO push_subscriptions (
    user_id,
    endpoint,
    p256dh,
    auth, logout_token_hash
)
VALUES (
    @user_id,
    @endpoint,
    @p256dh,
    @auth, @logout_token_hash
)
ON CONFLICT (endpoint) DO UPDATE SET
    p256dh = EXCLUDED.p256dh,
    auth = EXCLUDED.auth,
    revoked_at = NULL,
    updated_at = now(),
    logout_token_hash = EXCLUDED.logout_token_hash
WHERE push_subscriptions.user_id = EXCLUDED.user_id
RETURNING id;

-- name: RevokePushSubscription :exec
UPDATE push_subscriptions
SET
    revoked_at = now(),
    updated_at = now()
WHERE id = @id
    AND user_id = @user_id
    AND revoked_at IS NULL;
-- name: RevokePushBrowser :exec
UPDATE push_subscriptions SET revoked_at=now(),updated_at=now()
WHERE logout_token_hash = sqlc.narg('logout_token_hash') AND revoked_at IS NULL;
