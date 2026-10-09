# 0008 — Turnia staff are separate, and see aggregates only

**Status:** accepted · **Date:** 2026-10-09

## Context

Whoever runs Turnia needs a dashboard: how many pharmacies, who signed up this week, who is active,
who stopped using it. Every tenant table is behind row-level security (ADR 0002), so this is the
first thing that must deliberately read across pharmacies. Turnia is also a *processor* of the
pharmacies' employee data under GDPR: running the service doesn't entitle its operator to read
who works when, or who was on sick leave.

## Decision

**Three roles, two kinds of account.**

- Pharmacy accounts (`users`): `admin` and `employee`, scoped to one pharmacy, as before.
- Turnia staff (`staff` table): no pharmacy, own sign-in at `/api/v1/platform/auth/*`. Their access
  token has a different audience (`aud: platform` vs `aud: pharmacy`); each side's middleware
  rejects the other's tokens. Refresh cookies are scoped to their own path. The first staff
  account is created with `turnia staff create`; there is no API to create staff.

**Aggregates through functions, run as a role that can do nothing else.**

- The stats are `SECURITY DEFINER` SQL functions (`platform_stats(from, to)`,
  `platform_pharmacies()`) that return counts, dates and pharmacy names — never a person's name,
  email, or shift.
- They are owned by `turnia_stats_owner`: `NOLOGIN`, `BYPASSRLS`, `SELECT` on the tables the stats
  need. It can't log in; it lends its rights only to code inside those functions.
- `EXECUTE` is revoked from `PUBLIC` (Postgres grants it to everyone by default) and granted only to
  **`turnia_platform`**, the role the platform endpoints connect as, through their own connection
  pool. `turnia_platform` has no table privileges at all.
- `turnia_app`, the role every pharmacy request uses, can't execute them. A bug in a pharmacy
  endpoint can't reach cross-tenant data, and a bug in a platform endpoint can't read rows.
- Activity comes from `users.last_seen_at`, updated at token refresh at most once a day — enough
  for "active this week", and no tracking of what people look at.

Tried in psql, 2026-10-09: with the default grants, `turnia_app` could call the function and got
every pharmacy's counts; after `REVOKE … FROM PUBLIC` it gets `permission denied for function`;
`turnia_platform` gets the aggregates and is refused the table itself.

## Alternatives

- **A `role = 'staff'` value in `users`** — staff would need a pharmacy, or `pharmacy_id` would be
  nullable and every policy would need a special case. One token could then carry both meanings.
- **A `BYPASSRLS` role for the dashboard's connection** — simplest, and it can read every
  employee's shifts. One sloppy query away from breaking N1.
- **Product analytics (PostHog, Plausible)** — useful later for funnels; it doesn't answer "how many
  shifts did pharmacy X plan", and it would send data to a third party.

## Consequences

- Three login roles in production (`turnia_app`, `turnia_platform`, and the migration owner) and a
  second connection string for the API.
- A new figure on the dashboard means a new or changed function in a migration, reviewed for what
  it returns. That friction is the point.
- Helping a pharmacy with their actual data (support access) needs its own design: time-limited,
  granted by the pharmacy admin, audited (design §8).
- Staff accounts are the most powerful in the system; two-factor sign-in comes before billing.
