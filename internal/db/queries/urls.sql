-- name: CreateURL :one
INSERT INTO urls (original_url, short_url, user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetURLByID :one
SELECT * FROM urls WHERE id = $1 AND is_active = true;

-- name: GetURLByShortURL :one
SELECT * FROM urls WHERE short_url = $1 AND is_active = true;

-- name: UpdateURL :one
UPDATE urls SET original_url = $2, short_url = $3, updated_at = NOW() WHERE id = $1 AND is_active = true RETURNING *;

-- name: DeleteURL :exec
UPDATE urls SET is_active = false, updated_at = NOW() WHERE id = $1;

-- name: ListURLs :many
SELECT * FROM urls WHERE user_id = $1 AND is_active = true LIMIT $2 OFFSET $3;

-- name: GetURLCount :one
SELECT COUNT(*) FROM urls WHERE user_id = $1 AND is_active = true;

-- name: ListAllURLs :many
SELECT * FROM urls WHERE is_active = true LIMIT $1 OFFSET $2;

-- name: CountAllURLs :one
SELECT COUNT(*) FROM urls WHERE is_active = true;

-- name: ListAllURLsByDate :many
SELECT * FROM urls WHERE is_active = true ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountAllURLsByDate :one
SELECT COUNT(*) FROM urls WHERE is_active = true AND created_at BETWEEN $1 AND $2;

-- name: UpdateURLStatus :one
UPDATE urls SET is_active = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: CountActiveURLs :one
SELECT COUNT(*) FROM urls WHERE is_active = true;

-- name: CountInactiveURLs :one
SELECT COUNT(*) FROM urls WHERE is_active = false;

-- name: GetTopURLsByClicks :many
SELECT u.id, u.short_url, u.original_url, u.user_id, u.is_active, u.created_at, u.updated_at,
    COUNT(c.id) AS click_count
FROM urls u
LEFT JOIN clicks c ON c.url_id = u.id
WHERE u.is_active = true
GROUP BY u.id
ORDER BY click_count DESC
LIMIT $1;
