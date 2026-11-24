-- name: UpsertPushSubscription :one
INSERT INTO push_subscriptions (
    user_id,
    endpoint,
    p256dh,
    auth
)
VALUES (
    @user_id,
    @endpoint,
    @p256dh,
    @auth
)
ON CONFLICT (endpoint) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    p256dh = EXCLUDED.p256dh,
    auth = EXCLUDED.auth,
    revoked_at = NULL,
    updated_at = now()
RETURNING id;

-- name: RevokePushSubscription :exec
UPDATE push_subscriptions
SET
    revoked_at = now(),
    updated_at = now()
WHERE id = @id
    AND user_id = @user_id
    AND revoked_at IS NULL;