-- name: CreateIdentity :exec
INSERT INTO identities (id, user_id, provider, provider_user_id, password_hash)
VALUES ($1, $2, $3, $4, $5);

-- name: GetIdentitiesByUserID :many
SELECT id, user_id, provider, provider_user_id, password_hash
FROM identities
WHERE user_id = $1;
