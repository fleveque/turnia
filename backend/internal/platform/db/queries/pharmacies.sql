-- name: CreatePharmacy :one
INSERT INTO pharmacies (name)
VALUES ($1)
RETURNING *;
