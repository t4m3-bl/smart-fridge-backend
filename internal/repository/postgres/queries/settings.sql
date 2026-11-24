-- name: GetSettings :one
SELECT * FROM settings WHERE user_id = @user_id;

-- name: CreateDefaultSettings :exec
WITH created AS (
    INSERT INTO settings (user_id)
    VALUES (@user_id)
    ON CONFLICT (user_id) DO NOTHING
    RETURNING user_id
)
INSERT INTO indicators (id, user_id, color, max_days)
SELECT uuidv7(), created.user_id, defaults.color, defaults.max_days
FROM created
CROSS JOIN (VALUES
    ('#EF4444', 1),
    ('#EAB308', 3),
    ('#22C55E', NULL)
) AS defaults(color, max_days);

-- name: PatchSettings :one
UPDATE settings SET
    has_notifications = COALESCE(sqlc.narg('has_notifications'), has_notifications),
    notification_time = COALESCE(sqlc.narg('notification_time'), notification_time),
    timezone          = COALESCE(sqlc.narg('timezone'), timezone)
WHERE user_id = @user_id
RETURNING *;

-- name: ListIndicators :many
SELECT * FROM indicators WHERE user_id = @user_id ORDER BY max_days NULLS LAST, id;

-- name: DeleteIndicators :exec
DELETE FROM indicators WHERE user_id = @user_id;

-- name: CreateIndicator :exec
INSERT INTO indicators (id, user_id, color, max_days)
VALUES (@id, @user_id, @color, @max_days);
