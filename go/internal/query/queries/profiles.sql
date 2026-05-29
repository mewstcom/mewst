-- name: GetProfileByID :one
SELECT * FROM profiles WHERE id = $1 LIMIT 1;

-- name: GetProfileByAtname :one
SELECT * FROM profiles WHERE atname = $1 LIMIT 1;

-- name: ExistsProfileByAtname :one
SELECT EXISTS(
    SELECT 1 FROM profiles WHERE atname = $1
) AS exists;

-- name: UpdateProfileLastPostAt :exec
UPDATE profiles
SET last_post_at = $2, updated_at = NOW()
WHERE id = $1;

-- name: CreateProfile :one
INSERT INTO profiles (owner_type, atname, name, description, image_url, joined_at, avatar_kind, gravatar_email, gravatar_url, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
RETURNING *;
