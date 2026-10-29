-- +goose Up

-- Tenant isolation (ADR 0002). Every query the server makes runs in a
-- transaction that first sets app.pharmacy_id to the signed-in user's
-- pharmacy (db.WithTenant, milestone 3.3), and these policies hide every row
-- of any other pharmacy, whatever the query's WHERE says.
--
-- ENABLE turns the policies on; FORCE applies them to the tables' owner too,
-- which would otherwise be exempt. (A superuser is exempt from both, so the
-- tests connect as turnia_app.)
--
-- current_setting has no missing_ok argument here, on purpose: with no tenant
-- set, it raises an error instead of returning NULL, and a query without a
-- tenant fails instead of quietly seeing nothing. Once a transaction-local
-- setting has ended, it returns '', and ''::uuid fails too.
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- USING filters the rows a statement can see, update or delete. With no WITH
-- CHECK, the same condition also applies to the rows a statement writes: an
-- INSERT, or an UPDATE that moves a user, into another pharmacy is refused.
CREATE POLICY tenant_isolation ON users
    USING (pharmacy_id = current_setting('app.pharmacy_id')::uuid);

ALTER TABLE pharmacies ENABLE ROW LEVEL SECURITY;
ALTER TABLE pharmacies FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON pharmacies
    USING (id = current_setting('app.pharmacy_id')::uuid);

-- +goose Down

DROP POLICY tenant_isolation ON pharmacies;
ALTER TABLE pharmacies NO FORCE ROW LEVEL SECURITY;
ALTER TABLE pharmacies DISABLE ROW LEVEL SECURITY;

DROP POLICY tenant_isolation ON users;
ALTER TABLE users NO FORCE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
