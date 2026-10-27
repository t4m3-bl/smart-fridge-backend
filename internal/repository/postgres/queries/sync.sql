-- name: LockUser :one
SELECT id
FROM users
WHERE id = $1
FOR UPDATE;

-- name: GetMutationResult :one
SELECT seq, status, reason, message
FROM mutations
WHERE user_id = $1 AND id = $2;

-- name: CreateMutation :one
INSERT INTO mutations (id, user_id, type, product_id, payload, status, reason, message)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING seq;

-- name: ListAppliedMutations :many
SELECT seq, id, type, product_id, payload, created_at
FROM mutations
WHERE user_id = sqlc.arg(user_id)
  AND status = 'applied'
  AND seq > sqlc.arg(since)
ORDER BY seq
LIMIT sqlc.arg(max_rows);
