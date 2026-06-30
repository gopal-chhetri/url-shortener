-- name: CreateUser :one
INSERT INTO users (email, password_hash, first_name, last_name, role_id)
VALUES ($1, $2, $3, $4, (SELECT id FROM roles WHERE name = 'user' LIMIT 1))
RETURNING *;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email=$1 AND is_active=true;

-- name: GetUserById :one
SELECT *
FROM users
WHERE id=$1 AND is_active=true;

-- name: UpdateUserById :one
UPDATE users
SET first_name = $2, last_name = $3, email = $4, updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING *;

-- name: DeleteUserById :exec
UPDATE users
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: ListUsers :many
SELECT *
FROM users
WHERE is_active = true
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*)
FROM users
WHERE is_active = true;

-- name: CreateRole :one
INSERT INTO roles (name, description)
VALUES ($1, $2)
RETURNING id, name, description, created_at, updated_at;

-- name: GetRoleByName :one
SELECT id, name, description, created_at, updated_at
FROM roles
WHERE name = $1;

-- name: GetRoleByID :one
SELECT id, name, description, created_at, updated_at
FROM roles
WHERE id = $1;

-- name: GetRoleNameByID :one
SELECT name FROM roles WHERE id = $1;

-- name: ListRoles :many
SELECT id, name, description, created_at, updated_at
FROM roles
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountRoles :one
SELECT COUNT(*)
FROM roles;

-- name: UpdateUserRole :one
UPDATE users
SET role_id = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserStatus :one
UPDATE users
SET is_active = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListAllUsers :many
SELECT *
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAllUsers :one
SELECT COUNT(*) FROM users;
