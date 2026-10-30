-- Production's database, owned by the role that runs the migrations. Owning
-- it is what lets turnia_owner create tables in the public schema (since
-- Postgres 15, only the database's owner can) and the citext extension.
-- Development's databases are made by dev/postgres-init and dbtest instead.
CREATE DATABASE turnia OWNER turnia_owner;
