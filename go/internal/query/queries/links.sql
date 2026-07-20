-- name: GetLinkByCanonicalURL :one
SELECT * FROM links WHERE canonical_url = $1 LIMIT 1;

-- name: CreateLink :one
INSERT INTO links (canonical_url, domain, title, image_url, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW())
RETURNING *;
