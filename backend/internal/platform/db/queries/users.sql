-- name: CreateUser :one
INSERT INTO users (pharmacy_id, email, name, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListUsers :many
SELECT * FROM users
WHERE pharmacy_id = $1
ORDER BY name;
