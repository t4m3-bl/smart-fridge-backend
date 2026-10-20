-- name: CreateIdentity :exec
INSERT INTO identities (id, user_id, provider, provider_user_id, password_hash)
VALUES ($1, $2, $3, $4, $5);