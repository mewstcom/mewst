-- name: CreatePostLink :one
INSERT INTO post_links (post_id, link_id, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
RETURNING *;
