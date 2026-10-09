# 0002 — One schema for all pharmacies, isolated by row-level security

**Status:** accepted · **Date:** 2026-10-09

## Context

Every pharmacy is a tenant, and one pharmacy seeing another's employees or schedules is the worst
thing Turnia could do (design N1). Tenants are small (a handful of users, a few thousand shifts a
year) and there will be many of them.

## Decision

- **One database, one schema.** Every tenant table has a `pharmacy_id` column.
- **Postgres row-level security (RLS)** on every tenant table, with `FORCE ROW LEVEL SECURITY` so
  it also applies to the table owner. The policy compares `pharmacy_id` with a per-transaction
  setting: `current_setting('app.pharmacy_id')`.
- The application connects as **`turnia_app`**, a role that does not own the tables and cannot
  bypass RLS. Migrations run as the owner, from a separate connection string.
- Every request that touches tenant data runs inside **`db.WithTenant(ctx, pharmacyID, fn)`**: open
  a transaction, `set_config('app.pharmacy_id', …, true)` (the `true` makes it local to the
  transaction, so a pooled connection never carries it to the next request), run `fn`, commit.
- Queries still filter by `pharmacy_id` explicitly. RLS is the second lock, not the only one.
- The two moments with no tenant yet:
  - **login** looks a user up by email through one `SECURITY DEFINER` function that returns only
    what login needs;
  - **register** generates the new pharmacy's id in Go and uses it as the tenant for the
    transaction that creates the pharmacy and its admin — no bypass needed.

## Alternatives

- **Application filtering only** — one forgotten `WHERE` leaks data, and nothing notices.
- **Schema per tenant** — strong isolation, but migrations run N times, connection pooling gets
  harder, and cross-tenant operations (billing, support) need dynamic SQL. Overkill for tenants
  this small.
- **Database per tenant** — the same, more so.

## Consequences

- A test suite proves isolation: data written as pharmacy A is invisible as pharmacy B, even with a
  query that deliberately omits the `pharmacy_id` filter.
- Every tenant query runs in a transaction. That is a cost of one round trip per request, accepted.
- Anything that must cross tenants (a future admin console, billing jobs) needs its own role and is
  a deliberate, reviewed exception.
