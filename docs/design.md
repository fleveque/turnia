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

- **N1. A pharmacy never sees another pharmacy's data, and Turnia's own staff see figures, not
  people.** Enforced by the database (row-level security), not only by the application's queries.
  The platform dashboard reads aggregates through functions that return no employee's name, email
  or shifts; the only people it names are each pharmacy's admins, Turnia's customers.
  See [ADR 0002](decisions/0002-shared-schema-with-row-level-security.md) and
  [ADR 0008](decisions/0008-platform-staff-and-aggregate-stats.md).
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

- A **pharmacy** is a tenant. Everything else belongs to exactly one pharmacy, except Turnia's own
  staff.
- Three roles:

  | Role | Who | Can |
  |---|---|---|
  | **Turnia staff** (`staff`) | the people who run Turnia | see the platform dashboard: pharmacies, sign-ups, activity, usage — aggregates only; manage plans later |
  | **Pharmacy admin** (`admin`) | the owner or manager of a pharmacy | plan and edit shifts, manage employees, see everyone's hours and history, change pharmacy settings |
  | **Employee** (`employee`) | everyone else in the pharmacy | see their own week, hours and history, the team's shifts, and swap if enabled |

- **Staff are not users of a pharmacy.** They live in their own table, sign in at their own
  endpoint, and get a token no pharmacy endpoint accepts; a pharmacy token can't reach the platform
  endpoints either. The first staff account is created from the command line (`turnia staff
  create`), never through the API.
- A pharmacy **user** belongs to one pharmacy and has the role `admin` or `employee`. A pharmacy can
  have several admins.
- **Everyone in a pharmacy sees everyone's shifts** — who works when is how a team organises itself
  — but **only admins create, edit or delete them**. The single exception is a swap (below), which
  employees make between themselves and which the system carries out and records.
- What employees do **not** see about colleagues: their hours and contract, and the details of an
  absence. A colleague's sick leave shows as *Ausente* with no type or notes; the reason for an
  absence can be health data (GDPR, special category), and only the person and admins need it.
- **Shift swaps** (a per-pharmacy setting, off by default): an employee proposes exchanging one of
  their shifts for a colleague's; if the colleague accepts, both shifts change hands in one
  transaction. Swapped shifts are marked as such, and every step — proposed, accepted, declined,
  cancelled — is in the audit log. See [ADR 0007](decisions/0007-shift-swaps-between-employees.md).
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
| `pharmacies` | the tenant: name, time zone, default locale, `plan` (`free` for now; billing later), settings such as `swaps_enabled` |
| `users` | name, email, role, locale (`es`/`ca`), active, weekly contract hours, `last_seen_at` (day precision, for activity figures) |
| `staff` | Turnia's own people: name, email — no pharmacy, no RLS policy to fit into |
| `login_codes` | one sign-in or invitation email: user or staff, hashed code, hashed link token, expiry, attempts left, used at |
| `shift_types` | per pharmacy: name, colour, icon, `kind` (`regular`, `on_call`, `absence`), whether it counts as hours |
| `shifts` | type, assignee (nullable: an unassigned shift), start, end, notes, `last_changed_at` |
| `shift_swaps` | a swap proposal: who asks, which of their shifts, whom, which of theirs, status (`pending`, `accepted`, `declined`, `cancelled`, `expired`), message, times |
| `shift_events` | the audit log: shift, actor, action, before/after as JSON, the swap involved if any, time — append-only |
| `refresh_tokens` | hashed, rotated refresh tokens |

No passwords anywhere (ADR 0003). When Google sign-in arrives, a `user_identities` table (provider,
the provider's account id) links it to the existing user; the account was never the email.

Rules the database enforces, not the application:

- a shift ends after it starts;
- **one person cannot be in two shifts at the same time** — an exclusion constraint over
  `(user_id, tstzrange(starts_at, ends_at))`;
- every row of every tenant table is filtered by the current pharmacy (N1);
- every change to `shifts` writes a `shift_events` row, and `shift_events` cannot be updated or
  deleted by the application (N2); so does every status change of a `shift_swaps` row, on both
  shifts involved;
- a shift has at most one pending swap at a time (a partial unique index);
- the overlap constraint is checked at commit (`DEFERRABLE`) inside a swap, so exchanging two
  shifts doesn't trip over its own intermediate state — but a swap that would double-book either
  person still fails.

**Derived, not stored**: whether a shift falls on a weekend (and, later, a public holiday) is
computed from its local date and returned by the API as a flag, so the frontend does not
re-implement business rules. "Changed" means an admin edited the shift's times, assignee or type
after creating it; "swapped" means it changed hands through an accepted swap, with whom and when.
The two are separate badges.

**Hours**: for a person and a date range, the API returns minutes *worked* (before now), *planned*
(after now; a shift in progress is split at now), the *contract* expectation, and a breakdown by
shift kind. Weightings (a *guardia* hour counting differently) are a later formula over that
breakdown, not a schema change.

## 6. API

REST over JSON under `/api/v1`, errors as RFC 9457 `application/problem+json`.

| Method | Path | Who |
|---|---|---|
| GET | `/healthz` | anyone |
| POST | `/auth/register` — a pharmacy and its first admin; emails a code | anyone |
| POST | `/auth/code` — email me a sign-in code and link (same answer whether the email exists or not) | anyone |
| POST | `/auth/verify` — a code with its email, or a link token → access token + refresh cookie | anyone |
| POST | `/auth/refresh`, `/auth/logout` | refresh cookie |
| GET, PATCH | `/me` | signed in |
| GET | `/me/shifts?from&to`, `/me/hours?from&to` | signed in |
| GET | `/shift-types` | signed in |
| GET, POST | `/employees` — POST invites by email | admin |
| PATCH | `/employees/{id}` | admin |
| POST | `/employees/{id}/invitation` — send the invitation again | admin |
| GET | `/employees/{id}/hours?from&to` | admin |
| GET | `/shifts?from&to&user_id` — the whole team's shifts (absences redacted for employees) | signed in |
| POST | `/shifts` | admin |
| PATCH, DELETE | `/shifts/{id}` | admin |
| GET | `/shifts/{id}/history` | admin; an employee for their own shifts |
| GET | `/swaps?status` — swaps I'm part of (admin: all) | signed in |
| POST | `/swaps` — propose: my shift, their shift, a message | employee, if swaps are enabled |
| POST | `/swaps/{id}/accept`, `/swaps/{id}/decline` | the colleague asked |
| POST | `/swaps/{id}/cancel` | the employee who asked; an admin |
| GET, PATCH | `/pharmacy` — name, time zone, settings (`swaps_enabled`) | signed in / admin |
| POST | `/platform/auth/code`, `/platform/auth/verify`, `/platform/auth/refresh`, `/platform/auth/logout` | staff |
| GET | `/platform/stats?from&to` — totals and weekly series: pharmacies, sign-ups, active users, shifts created, swaps, plans | staff |
| GET | `/platform/pharmacies` — one row per pharmacy: name, admins' contact emails, created, plan, number of employees, last activity, shifts this month | staff |

Platform endpoints are served by the same binary but use their own connection pool, as a database
role that can call the stats functions and nothing else (ADR 0008).

Date ranges are capped (about two months) so no request scans a pharmacy's whole history.

## 7. Frontend

- **Mi semana** (my week) is the home screen. On a phone: seven day cards stacked vertically, today
  highlighted, previous/next week by swipe or buttons. From a tablet up: a seven-column grid.
- **A shift card** shows the type's colour, icon and name, the time range and duration. Badges:
  *Guardia*, *Fin de semana*, *Modificado* (tapping it shows before and after), *Intercambiado*
  (with whom), *Intercambio pendiente*. Absences are a hatched full-day block. A legend is one tap
  away. Colours come from one set of design tokens.
- **Equipo** (team) — the whole pharmacy's week, read-only for employees: a grid of people × days
  using the same shift card, your own row first. On a phone, one day at a time with the team
  stacked.
- **Swaps** (when enabled) — from one of your shifts, "Proponer intercambio" lists colleagues'
  shifts you could take without overlapping; the colleague sees the request in a small inbox and
  accepts or declines with one tap.
- **Hours** — a compact bar on Mi semana: worked, planned, contract.
- **Admin** — the Equipo grid, made editable: a form to create and edit shifts, an hours column per
  employee, the employees list, a shift's history, pharmacy settings (swaps on or off).
- **Plataforma** (Turnia staff) — a separate section with its own sign-in, loaded only for staff: a
  row of headline figures (pharmacies, active users this week, shifts planned this week), sign-ups
  and activity by week, plan distribution, and the pharmacies table, sortable by last activity so
  pharmacies that stopped using Turnia stand out. Desktop first; it still works on a phone.
- **Offline** — the service worker keeps the last-seen week and hours, so the screen works in a
  basement stockroom.
- **Signing in** — enter your email, then type the 6-digit code or tap the link in the email; the
  code is what works in the installed iPhone app. Invited employees start from the invitation email.
  You stay signed in for months on your own phone; "Cerrar sesión" on a shared computer.
- **Language** — Spanish and Catalan. The user's choice is stored on their account; before login,
  the browser's language decides (`ca*` → Catalan, otherwise Spanish).

## 8. Out of scope for the MVP, designed for

In rough order of value and effort:

1. Editing shift types (name, colour, icon) — data plus a form, thanks to the tokens.
2. Public holidays (national, Catalan and other regions, local) as a flag and a badge.
3. Draft and published weeks, and "changes since you last looked".
4. Monthly and yearly hours, exports for the payroll advisor (*gestoría*), weightings per kind.
5. Push notifications when your shifts change.
6. More swap modes: requiring an admin's approval as a setting, and giving a shift away without
   taking one back (a *cover* request).
7. Billing per pharmacy with Lemon Squeezy (`pharmacies.plan` is the hook); staff change a
   pharmacy's plan from the dashboard. Two-factor sign-in (TOTP) for staff before any of that.
8. Sign in with Google, linked to existing accounts (ADR 0003), once tried in the installed app on an
   iPhone.
9. Support access: a pharmacy admin grants Turnia staff temporary access to their pharmacy's data
   to resolve an issue — time-limited, visible to the pharmacy, and audited.
10. Import from the Excel file people already have.
11. The *IA* in Turn*IA*: suggesting a schedule from constraints (contract hours, *guardia* rota,
    holidays).
