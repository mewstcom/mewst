-- name: GetUserProfileByUserID :one
SELECT * FROM user_profiles WHERE user_id = $1 LIMIT 1;

-- name: GetUserProfileByProfileID :one
SELECT * FROM user_profiles WHERE profile_id = $1 LIMIT 1;

-- name: CreateUserProfile :one
INSERT INTO user_profiles (user_id, profile_id, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
RETURNING *;
