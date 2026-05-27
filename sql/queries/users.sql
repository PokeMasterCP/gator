-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, name)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetUserByName :one
SELECT *
FROM users
WHERE name = $1
LIMIT 1;

-- name: ClearUsersTable :exec
DELETE FROM users;

-- name: GetAllUsers :many
SELECT name FROM users;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1
LIMIT 1;
