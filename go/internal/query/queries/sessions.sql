-- name: GetSessionByToken :one
SELECT * FROM sessions WHERE token = $1 LIMIT 1;

-- name: GetAuthByToken :one
-- Resolves a session token to the associated actor, user, and profile in a
-- single JOIN so authenticated-page middleware can avoid issuing four separate
-- queries (session, actor, user, profile) per request.
-- [Ja] セッショントークンに紐づく actor / user / profile を 1 度の JOIN で
-- 取得する。認証後ページの middleware が 1 リクエストあたり 4 クエリ
-- (session, actor, user, profile) を発行するのを避けるため。
SELECT
    sqlc.embed(actors),
    sqlc.embed(users),
    sqlc.embed(profiles)
FROM sessions
INNER JOIN actors ON actors.id = sessions.actor_id
INNER JOIN users ON users.id = actors.user_id
INNER JOIN profiles ON profiles.id = actors.profile_id
WHERE sessions.token = $1
LIMIT 1;

-- name: CreateSession :one
INSERT INTO sessions (actor_id, token, ip_address, user_agent, signed_in_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW())
RETURNING *;

-- name: DeleteSessionByToken :exec
DELETE FROM sessions
WHERE token = $1;
