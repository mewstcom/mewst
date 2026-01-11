-- name: CreateEmailConfirmation :one
INSERT INTO email_confirmations (email, event, code, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
RETURNING id, email, event, code, succeeded_at, created_at, updated_at;

-- name: GetEmailConfirmationByID :one
SELECT id, email, event, code, succeeded_at, created_at, updated_at
FROM email_confirmations
WHERE id = $1
LIMIT 1;

-- name: GetActiveEmailConfirmationByID :one
SELECT id, email, event, code, succeeded_at, created_at, updated_at
FROM email_confirmations
WHERE id = $1
  AND succeeded_at IS NULL
  AND created_at > NOW() - INTERVAL '15 minutes'
LIMIT 1;

-- name: GetSucceededEmailConfirmationByID :one
SELECT id, email, event, code, succeeded_at, created_at, updated_at
FROM email_confirmations
WHERE id = $1
  AND succeeded_at IS NOT NULL
LIMIT 1;

-- name: MarkEmailConfirmationAsSucceeded :exec
UPDATE email_confirmations
SET succeeded_at = NOW(), updated_at = NOW()
WHERE id = $1;
