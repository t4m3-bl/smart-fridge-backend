-- name: CreateUser :exec
INSERT INTO users (id, email)
VALUES ($1, $2);

-- name: GetUserByEmail :one
SELECT id, email
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email
FROM users
WHERE id = $1;

-- name: GetUserByIdentity :one
SELECT u.id, u.email
FROM users u
JOIN identities i ON u.id = i.user_id
WHERE i.provider = $1 AND i.provider_user_id = $2;
