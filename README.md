# Turnia

**Shift scheduling for small pharmacies.** Employees see their week on their phone, offline included;
the owner plans shifts, sees everyone's hours, and knows who changed what. Spanish and Catalan.

Turnia replaces the Excel file pinned to the stockroom wall. It is open source (AGPL-3.0) and is
being built in public, in small pull requests, by someone learning Go and React while doing it —
see [Learning](#learning).

> **Status:** milestone 1 — the Go server skeleton. `make run` serves `/api/v1/healthz`.

## What it does (MVP)

- **Mi semana** — your shifts for the week, at a glance: type (morning, afternoon, *guardia*,
  absence…), weekend, changed. Installable from the browser (PWA), works offline.
- **Hours** — worked, still planned, and what your contract expects.
- **Equipo** — the whole team's week, so everyone knows who's in. Only admins edit shifts.
- **Swaps** — if the pharmacy turns it on, two employees can exchange shifts when both accept;
  the swap shows on both shifts and is on record.
- **Admin** — plan the week for everyone, manage employees, see any shift's history.
- **Plataforma** — for the people running Turnia: pharmacies, sign-ups, activity and usage, as
  figures only. Turnia staff never see a pharmacy's employees or shifts.

Three roles: **Turnia staff**, **pharmacy admins**, **employees**.
- **Every shift change is recorded**: who, when, before and after.
- **Each pharmacy's data is isolated** by the database itself, not only by the application.
- **No passwords**: sign in with a code or link sent by email; employees are invited by their admin.

The full picture is in [docs/design.md](docs/design.md); the reasons behind the choices are in
[docs/decisions/](docs/decisions/README.md).

## Stack

| | |
|---|---|
| Backend | Go (standard library HTTP), PostgreSQL 18, pgx, sqlc, goose; email through Resend |
| Frontend | React, TypeScript, Vite, TanStack Query, Tailwind CSS — as a PWA |
| Deploy | Kamal to a Hetzner VPS on every merge to `main`: one container (Go binary with the PWA embedded) + Postgres |

## Development

Requirements: Go 1.27+, Node 24+, Docker.

```sh
make            # list every task
make db-up      # start Postgres 18 on localhost:5433 (databases turnia and turnia_test)
make db-psql    # open psql on it
```

Postgres runs on **5433** so it doesn't collide with a Postgres installed on the host.

## Roadmap

Each milestone is one or more small pull requests, each with a code walkthrough, and the milestone
ends with a lesson. From milestone 4, merging
a pull request deploys it.

| # | Milestone | |
|---|---|---|
| 0 | Repository, design, decisions | ✓ |
| 1 | Go server skeleton: config, logging, health check, errors, graceful shutdown | ◐ |
| 2 | Postgres: pool, migrations, sqlc | |
| 3 | Tenancy with row-level security | |
| 4 | Deploy: image, Kamal, merge-to-deploy, backups — live on a beta host | |
| 5 | Auth I: email codes and links, register, access tokens, `/me` | |
| 6 | Auth II: refresh tokens, logout, rate limits | |
| 7 | Employees, invitations, pharmacy settings | |
| 8 | Shift types and shifts — the team sees all, admins edit | |
| 9 | Audit log for shifts | |
| 10 | Hours counters | |
| 11 | Shift swaps between employees | |
| 12 | Platform: Turnia staff, sign-in and aggregate stats | |
| 13 | Seed data and the OpenAPI contract | |
| 14 | Frontend scaffold — embedded in the Go binary, live from here | |
| 15 | Spanish and Catalan | |
| 16 | API layer and mocks | |
| 17 | **Mi semana** — the visual weekly calendar | |
| 18 | **Equipo** — the team's week | |
| 19 | Hours widget and change details | |
| 20 | Frontend auth against the real API | |
| 21 | PWA: install, offline, updates | |
| 22 | Swaps: propose, accept, decline | |
| 23 | Admin: plan the week | |
| 24 | Admin: employees, history, settings | |
| 25 | **Plataforma** — the staff dashboard | |
| 26 | Launch: product domain, uptime alerts, first restore drill | |

After the MVP: editable shift types, public holidays, draft/published weeks, monthly hours and
exports, push notifications, more swap modes, Google sign-in, billing, Excel import, and the *IA* in Turn*IA* —
suggested schedules. See [design §8](docs/design.md#8-out-of-scope-for-the-mvp-designed-for).

## Learning

Turnia doubles as a way to learn Go and React properly. Every milestone ships a **lesson** — what
surprised me and why — and a **line-by-line walkthrough** of the code it added. They are indexed in
[docs/lessons/](docs/lessons/README.md).

## License

[AGPL-3.0](LICENSE). You can use, modify and self-host Turnia; if you offer a modified version to
others over a network, you share your changes under the same licence.
