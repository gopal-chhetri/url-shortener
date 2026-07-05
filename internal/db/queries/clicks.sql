-- name: CreateClick :one
INSERT INTO clicks (user_id, url_id, device, browser, latitude, longitude, ip_address, country, city)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetClickByID :one
SELECT * FROM clicks WHERE id = $1;

-- name: GetClicksByURLID :many
SELECT * FROM clicks WHERE url_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetClickCountByURLID :one
SELECT COUNT(*) FROM clicks WHERE url_id = $1;

-- name: GetClicksByUserID :many
SELECT * FROM clicks WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetClickCountByUserID :one
SELECT COUNT(*) FROM clicks WHERE user_id = $1;

-- name: GetRecentClicks :many
SELECT * FROM clicks ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: GetClickStatsByURLID :one
SELECT 
    COUNT(*) as total_clicks,
    COUNT(DISTINCT device) as unique_devices,
    COUNT(DISTINCT browser) as unique_browsers
FROM clicks WHERE url_id = $1;

-- name: GetClickStatsByDateRange :many
SELECT 
    DATE(created_at) as date,
    COUNT(*) as click_count
FROM clicks 
WHERE url_id = $1 AND created_at BETWEEN $2 AND $3
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- name: GetDeviceStatsByURLID :many
SELECT 
    COALESCE(device, 'Unknown') AS device,
    COUNT(*) AS count
FROM clicks
WHERE url_id = $1
GROUP BY device
ORDER BY count DESC;

-- name: GetBrowserStatsByURLID :many
SELECT 
    COALESCE(browser, 'Unknown') AS browser,
    COUNT(*) AS count
FROM clicks
WHERE url_id = $1
GROUP BY browser
ORDER BY count DESC;

-- name: CountAllClicks :one
SELECT COUNT(*) FROM clicks;

-- name: GetClickCountsByURLIDs :many
SELECT url_id, COUNT(*)::bigint AS click_count
FROM clicks
WHERE url_id = ANY($1::uuid[])
GROUP BY url_id;

-- name: GetGeoStatsByURLID :many
SELECT 
    COALESCE(country, 'Unknown') AS country,
    COUNT(*) AS count
FROM clicks
WHERE url_id = $1
GROUP BY country
ORDER BY count DESC;
