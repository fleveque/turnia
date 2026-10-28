-- +goose Up

-- The server connects as turnia_app (ADR 0002), which reads and writes rows
-- and nothing else: no CREATE, ALTER, DROP or TRUNCATE. Each migration that
-- adds a table grants on it here, explicitly, so what the app may do to a
-- table is in the same pull request as the table.
GRANT SELECT, INSERT, UPDATE, DELETE ON pharmacies, users TO turnia_app;

-- +goose Down

REVOKE ALL ON pharmacies, users FROM turnia_app;
