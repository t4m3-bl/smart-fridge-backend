-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, refresh_token_hash, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: RevokeActiveSession :one
UPDATE sessions
SET revoked_at = now()
WHERE refresh_token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > now()
RETURNING user_id;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = now()
WHERE refresh_token_hash = $1
  AND revoked_at IS NULL;
