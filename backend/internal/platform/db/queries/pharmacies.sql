-- The id comes from Go, not the column's default: as turnia_app, a pharmacy
-- can only be inserted in a transaction whose tenant is already its id
-- (db.WithTenant, migration 00003), so the id must be known first.
-- name: CreatePharmacy :one
INSERT INTO pharmacies (id, name)
VALUES ($1, $2)
RETURNING *;
