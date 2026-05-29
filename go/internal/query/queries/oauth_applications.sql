-- name: GetOauthApplicationByUID :one
SELECT * FROM oauth_applications WHERE uid = $1 LIMIT 1;
