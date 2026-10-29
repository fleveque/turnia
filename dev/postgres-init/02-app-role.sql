-- The role the server connects as (ADR 0002). It doesn't own the tables, so
-- it can't change the schema or bypass row-level security; migrations grant
-- it what it needs, table by table.
--
-- Roles belong to the whole server, not one database, so this isn't a
-- migration. It runs when the volume is first created, and again on every
-- `make db-up` (and in CI), so it must be safe to run twice. The password is
-- for development only; production's is in Bitwarden (milestone 4).
DO $$
BEGIN
    CREATE ROLE turnia_app LOGIN PASSWORD 'turnia_app';
EXCEPTION WHEN duplicate_object THEN
    NULL;
END
$$;
