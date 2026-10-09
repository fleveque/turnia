# Turnia — design

What Turnia is, what it must never do, and how it is built. Decisions with a story behind them have
their own record in [decisions/](decisions/README.md); this document is the map.

---

## 1. The problem

A small pharmacy has three to fifteen people and a schedule that changes every week: morning and
afternoon shifts, split shifts, the *guardia* (the night or holiday on-call duty every pharmacy takes
in turn), holidays, sick days, swaps. Today that schedule lives in an Excel file on the owner's
computer, printed and pinned to a wall, or photographed and sent to a WhatsApp group. Employees ask
"what am I doing on Thursday?" several times a week, and nobody can say who changed what.

Turnia replaces the spreadsheet with:

- **for employees** — their week on their phone, at any moment, even without coverage;
- **for the owner (admin)** — one place to plan shifts, see everyone's hours, and know every change
  that was made and by whom.

## 2. Non-negotiables

These are not traded for convenience. Later sections refer to them as N1–N6.

- **N1. A pharmacy never sees another pharmacy's data.** Enforced by the database (row-level
  security), not only by the application's queries. See [ADR 0002](decisions/0002-shared-schema-with-row-level-security.md).
- **N2. No shift changes without a record.** Every create, update and delete of a shift writes an
  audit event with who, when, before and after — enforced by a database trigger, so no code path
  can skip it. See [ADR 0005](decisions/0005-audit-shifts-with-a-database-trigger.md).
- **N3. An employee sees their week in two taps or fewer, on a phone, offline included.** Mobile is
  the primary target, desktop the adaptation.
- **N4. Very easy and very visual.** A shift's type (morning, *guardia*, absence…), whether it falls
  on a weekend, and whether it was changed must be readable at a glance — and never by colour
  alone.
- **N5. Spanish and Catalan from the first version.** No user-facing string is hardcoded; both
  locales ship complete or not at all.
- **N6. Open source.** AGPL-3.0 ([ADR 0006](decisions/0006-agpl-3.0.md)). No feature depends on
  something only the hosted version can do.

## 3. Users and tenants

- A **pharmacy** is a tenant. Everything else belongs to exactly one pharmacy.
- A **user** belongs to one pharmacy and has a role: `admin` (plans shifts, manages employees) or
  `employee` (sees their own shifts and hours).
- Email is unique across Turnia, so logging in needs no pharmacy code. An employee working in two
  pharmacies is out of scope for the MVP; when it arrives, a `memberships` table replaces
  `users.pharmacy_id` without changing how tenancy is enforced.

## 4. Architecture

```
            phone / desktop browser (installed PWA)
                          │  HTTPS, one origin
                          ▼
        ┌──────────────── Caddy ────────────────┐
        │  /          → static PWA (React build) │
        │  /api/*     → turnia serve (Go)        │
        └────────────────────┬───────────────────┘
                             │ pgx, as role turnia_app (RLS applies)
                             ▼
                       PostgreSQL 18
```

- **Backend** — one Go binary, `turnia`, with subcommands `serve`, `migrate` and `seed`. Standard
  library HTTP server; pgx for Postgres; SQL written by hand and turned into typed Go by sqlc;
  migrations by goose, embedded in the binary. See [ADR 0001](decisions/0001-stack.md).
- **Frontend** — React + TypeScript, built by Vite, delivered as a Progressive Web App: installable
  from the browser, no app store. Server state through TanStack Query; types generated from the
  API's OpenAPI contract.
- **One origin.** The app and the API are served from the same host, so the refresh-token cookie is
  first-party and there is no CORS in production. See [ADR 0003](decisions/0003-auth-tokens-on-one-origin.md).
- **Deploy** — a single Hetzner VM running Docker Compose: Caddy, the API, Postgres, nightly
  backups. See [ADR 0004](decisions/0004-one-hetzner-vm.md).

### Repository layout

```
backend/                 Go module
  cmd/turnia/            the binary
  internal/              packages by feature: auth, employee, shift, … plus platform/ (db, httpx)
  db/migrations/         goose migrations (embedded)
  db/queries/            SQL that sqlc turns into Go
  api/openapi.yaml       the API contract; the frontend's types come from it
frontend/                Vite + React + TypeScript PWA
deploy/                  production compose file, Caddyfile, backups
docs/                    this design, decisions, lessons
```

Packages are organised **by feature** (`auth`, `shift`) rather than by layer (`handlers`,
`services`, `repositories`): a change to shifts touches one directory.

## 5. Data model

All ids are UUIDv7 (Postgres 18's `uuidv7()`): unique without coordination, and ordered by creation
time, so they index well. All timestamps are `timestamptz`; a pharmacy has a time zone
(`Europe/Madrid` by default) used to decide what "Saturday" or "today" means.

| Table | Purpose |
|---|---|
| `pharmacies` | the tenant: name, time zone, default locale, `plan` (`free` for now; billing later) |
| `users` | name, email, password hash, role, locale (`es`/`ca`), active, weekly contract hours |
| `shift_types` | per pharmacy: name, colour, icon, `kind` (`regular`, `on_call`, `absence`), whether it counts as hours |
| `shifts` | type, assignee (nullable: an unassigned shift), start, end, notes, `last_changed_at` |
| `shift_events` | the audit log: shift, actor, action, before/after as JSON, time — append-only |
| `refresh_tokens` | hashed, rotated refresh tokens |

Rules the database enforces, not the application:

- a shift ends after it starts;
- **one person cannot be in two shifts at the same time** — an exclusion constraint over
  `(user_id, tstzrange(starts_at, ends_at))`;
- every row of every tenant table is filtered by the current pharmacy (N1);
- every change to `shifts` writes a `shift_events` row, and `shift_events` cannot be updated or
  deleted by the application (N2).

**Derived, not stored**: whether a shift falls on a weekend (and, later, a public holiday) is
computed from its local date and returned by the API as a flag, so the frontend does not
re-implement business rules. "Changed" means the shift's times, assignee or type were edited after
it was created.

**Hours**: for a person and a date range, the API returns minutes *worked* (before now), *planned*
(after now; a shift in progress is split at now), the *contract* expectation, and a breakdown by
shift kind. Weightings (a *guardia* hour counting differently) are a later formula over that
breakdown, not a schema change.

## 6. API

REST over JSON under `/api/v1`, errors as RFC 9457 `application/problem+json`.

| Method | Path | Who |
|---|---|---|
| GET | `/healthz` | anyone |
| POST | `/auth/register` — a pharmacy and its first admin | anyone |
| POST | `/auth/login`, `/auth/refresh`, `/auth/logout` | anyone / refresh cookie |
| GET, PATCH | `/me` | signed in |
| GET | `/me/shifts?from&to`, `/me/hours?from&to` | signed in |
| GET | `/shift-types` | signed in |
| GET, POST | `/employees` | admin |
| PATCH | `/employees/{id}` | admin |
| GET | `/employees/{id}/hours?from&to` | admin |
| GET, POST | `/shifts` | admin |
| PATCH, DELETE | `/shifts/{id}` | admin |
| GET | `/shifts/{id}/history` | admin; an employee for their own shifts |

Date ranges are capped (about two months) so no request scans a pharmacy's whole history.

## 7. Frontend

- **Mi semana** (my week) is the home screen. On a phone: seven day cards stacked vertically, today
  highlighted, previous/next week by swipe or buttons. From a tablet up: a seven-column grid.
- **A shift card** shows the type's colour, icon and name, the time range and duration. Badges:
  *Guardia*, *Fin de semana*, *Modificado* (tapping it shows before and after). Absences are a
  hatched full-day block. A legend is one tap away. Colours come from one set of design tokens.
- **Hours** — a compact bar on Mi semana: worked, planned, contract.
- **Admin** — a week grid of employees × days using the same shift card, a form to create and edit
  shifts, an hours column per employee, the employees list, a shift's history.
- **Offline** — the service worker keeps the last-seen week and hours, so the screen works in a
  basement stockroom.
- **Language** — Spanish and Catalan. The user's choice is stored on their account; before login,
  the browser's language decides (`ca*` → Catalan, otherwise Spanish).

## 8. Out of scope for the MVP, designed for

In rough order of value and effort:

1. Editing shift types (name, colour, icon) — data plus a form, thanks to the tokens.
2. Public holidays (national, Catalan and other regions, local) as a flag and a badge.
3. Draft and published weeks, and "changes since you last looked".
4. Monthly and yearly hours, exports for the payroll advisor (*gestoría*), weightings per kind.
5. Push notifications when your shifts change.
6. Shift swaps between employees with admin approval — built on the audit log.
7. Billing per pharmacy with Lemon Squeezy (`pharmacies.plan` is the hook).
8. Import from the Excel file people already have.
9. The *IA* in Turn*IA*: suggesting a schedule from constraints (contract hours, *guardia* rota,
   holidays).
