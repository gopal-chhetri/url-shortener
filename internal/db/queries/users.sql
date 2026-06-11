-- name: CreateUser :one
INSERT INTO users (email, password_hash, first_name, last_name)
VALUES ($1, $2, $3, $4)
RETURNING id, email, first_name, last_name, role_id, is_active, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, first_name, last_name, role_id, is_active, created_at, updated_at
FROM users
WHERE email=$1 AND is_active=true;

-- name: GetUserById :one
SELECT id, email, first_name, last_name, role_id, is_active, created_at, updated_at
FROM users
WHERE id=$1 AND is_active=true;

-- name: UpdateUserById :one
UPDATE users
SET first_name = $2, last_name = $3, email = $4, updated_at = NOW()
WHERE id = $1 AND is_active = true
RETURNING id, email, first_name, last_name, role_id, is_active, created_at, updated_at;

-- name: DeleteUserById :exec
UPDATE users
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, first_name, last_name, role_id, is_active, created_at, updated_at
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

-- name: ListRoles :many
SELECT id, name, description, created_at, updated_at
FROM roles
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountRoles :one
SELECT COUNT(*)
FROM roles;

