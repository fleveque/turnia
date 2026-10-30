-- The two roles Turnia connects as (ADR 0002). Neither is a superuser, so
-- row-level security applies to both:
--
--   turnia_owner owns the database and the tables, and runs the migrations.
--                FORCE ROW LEVEL SECURITY keeps it inside the policies too.
--   turnia_app   is what the server connects as. It doesn't own the tables,
--                and migrations grant it row access, table by table.
--
-- Roles belong to the whole server, not one database, so this isn't a
-- migration. The same file runs everywhere: on production's Postgres when
-- its volume is first created, and in development and CI, where every
-- `make db-up` and every run repeats it, so it must be safe to run twice.
--
-- The passwords come from the environment, and no password is in this file.
-- If one isn't set, psql leaves the :'…' below as it is, and the statement
-- is a syntax error: the script stops instead of creating a role without one.
\getenv owner_password TURNIA_OWNER_PASSWORD
\getenv app_password TURNIA_APP_PASSWORD

-- CREATE ROLE can't take a psql variable inside a DO block, so each SELECT
-- builds the statement, %L quoting the password, and \gexec runs what it
-- returns. With the role already there, it returns no rows and runs nothing.
SELECT format('CREATE ROLE turnia_owner LOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE PASSWORD %L', :'owner_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'turnia_owner')
\gexec

SELECT format('CREATE ROLE turnia_app LOGIN PASSWORD %L', :'app_password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'turnia_app')
\gexec
