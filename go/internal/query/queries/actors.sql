-- name: GetActorByID :one
SELECT id, user_id, profile_id, created_at, updated_at
FROM actors
WHERE id = $1
LIMIT 1;

-- name: GetActorByUserID :one
SELECT id, user_id, profile_id, created_at, updated_at
FROM actors
WHERE user_id = $1
LIMIT 1;
