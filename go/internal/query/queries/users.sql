-- name: GetUserByID :one
SELECT id, email, password_digest, locale, time_zone, signed_up_at, created_at, updated_at
FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, email, password_digest, locale, time_zone, signed_up_at, created_at, updated_at
FROM users
WHERE email = $1
LIMIT 1;

-- name: GetUserByEmailForSignIn :one
SELECT id, email, password_digest
FROM users
WHERE email = $1
LIMIT 1;

-- name: UpdatePasswordByEmail :exec
UPDATE users
SET password_digest = $2, updated_at = NOW()
WHERE email = $1;

-- name: ExistsUserByEmail :one
SELECT EXISTS(
    SELECT 1 FROM users WHERE email = $1
) AS exists;

-- name: CreateUser :one
INSERT INTO users (email, password_digest, locale, time_zone, signed_up_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW())
RETURNING id, email, password_digest, locale, time_zone, signed_up_at, created_at, updated_at;
