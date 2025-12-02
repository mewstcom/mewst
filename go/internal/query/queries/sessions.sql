-- name: GetSessionByToken :one
SELECT id, actor_id, token, ip_address, user_agent, signed_in_at, created_at, updated_at
FROM sessions
WHERE token = $1
LIMIT 1;

-- name: CreateSession :one
INSERT INTO sessions (actor_id, token, ip_address, user_agent, signed_in_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW())
RETURNING id, actor_id, token, ip_address, user_agent, signed_in_at, created_at, updated_at;

-- name: DeleteSessionByToken :exec
DELETE FROM sessions
WHERE token = $1;
