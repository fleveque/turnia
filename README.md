# Turnia

**Shift scheduling for small pharmacies.** Employees see their week on their phone, offline included;
the owner plans shifts, sees everyone's hours, and knows who changed what. Spanish and Catalan.

Turnia replaces the Excel file pinned to the stockroom wall. It is open source (AGPL-3.0) and is
being built in public, in small pull requests, by someone learning Go and React while doing it —
see [Learning](#learning).

> **Status:** milestone 0 — design and repository. Nothing runs yet beyond the development database.

## What it does (MVP)

- **Mi semana** — your shifts for the week, at a glance: type (morning, afternoon, *guardia*,
  absence…), weekend, changed. Installable from the browser (PWA), works offline.
- **Hours** — worked, still planned, and what your contract expects.
- **Admin** — plan the week for everyone, manage employees, see any shift's history.
- **Every shift change is recorded**: who, when, before and after.
- **Each pharmacy's data is isolated** by the database itself, not only by the application.

The full picture is in [docs/design.md](docs/design.md); the reasons behind the choices are in
[docs/decisions/](docs/decisions/README.md).

## Stack

| | |
|---|---|
| Backend | Go (standard library HTTP), PostgreSQL 18, pgx, sqlc, goose |
| Frontend | React, TypeScript, Vite, TanStack Query, Tailwind CSS — as a PWA |
| Deploy | One Hetzner VM: Caddy + the Go binary + Postgres, with Docker Compose |

## Development

Requirements: Go 1.27+, Node 24+, Docker.

```sh
make            # list every task
make db-up      # start Postgres 18 on localhost:5433 (databases turnia and turnia_test)
make db-psql    # open psql on it
```

Postgres runs on **5433** so it doesn't collide with a Postgres installed on the host.

## Roadmap

Each milestone is one pull request, with a lesson and a code walkthrough.

| # | Milestone | |
|---|---|---|
| 0 | Repository, design, decisions | ◐ |
| 1 | Go server skeleton: config, logging, health check, errors, graceful shutdown | |
| 2 | Postgres: pool, migrations, sqlc | |
| 3 | Tenancy with row-level security | |
| 4 | Auth I: register, login, access tokens, `/me` | |
| 5 | Auth II: refresh tokens, logout, rate limit | |
| 6 | Employees | |
| 7 | Shift types and shifts | |
| 8 | Audit log for shifts | |
| 9 | Hours counters | |
| 10 | Seed data and the OpenAPI contract | |
| 11 | Frontend scaffold | |
| 12 | Spanish and Catalan | |
| 13 | API layer and mocks | |
| 14 | **Mi semana** — the visual weekly calendar | |
| 15 | Hours widget and change details | |
| 16 | Frontend auth against the real API | |
| 17 | PWA: install, offline, updates | |
| 18 | Admin: plan the week | |
| 19 | Admin: employees and history | |
| 20 | Deploy | |

After the MVP: editable shift types, public holidays, draft/published weeks, monthly hours and
exports, push notifications, shift swaps, billing, Excel import, and the *IA* in Turn*IA* —
suggested schedules. See [design §8](docs/design.md#8-out-of-scope-for-the-mvp-designed-for).

## Learning

Turnia doubles as a way to learn Go and React properly. Every milestone ships a **lesson** — what
surprised me and why — and a **line-by-line walkthrough** of the code it added. They are indexed in
[docs/lessons/](docs/lessons/README.md).

## License

[AGPL-3.0](LICENSE). You can use, modify and self-host Turnia; if you offer a modified version to
others over a network, you share your changes under the same licence.
