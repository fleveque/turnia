# 0001 — Go standard library, pgx, sqlc and goose; React PWA with Vite

**Status:** accepted · **Date:** 2026-10-09

## Context

Turnia is a small CRUD-plus-rules application: a few tables, a dozen endpoints, strong guarantees
about who can see what. It is also how its author learns Go and React, so every dependency has to
earn its place: each one is something to understand, not just to use.

## Decision

**Backend**

- **`net/http` from the standard library, no web framework.** Since Go 1.22 the `ServeMux`
  matches methods and path parameters (`GET /shifts/{id}`), which was the main reason to reach for
  chi or gin. Middleware is a function from `http.Handler` to `http.Handler`.
- **pgx** (v5) as the Postgres driver, used directly rather than through `database/sql`: native
  Postgres types (`tstzrange`, `jsonb`, UUIDs), a connection pool, and Postgres error codes we need
  (the exclusion constraint's `23P01` becomes a 409).
- **sqlc**: SQL is written by hand in `db/queries/*.sql`; sqlc generates typed Go functions from
  it. No ORM. Every query that touches tenant data is plain to read and review, which matters
  because tenant isolation (ADR 0002) is the property we can least afford to get wrong.
- **goose** for migrations, plain SQL files embedded in the binary; `turnia migrate` runs them.
  Already familiar from quantic-agent-go.
- sqlc and goose are pinned as `tool` directives in `go.mod` and run with `go tool`: no global
  installs, same versions in CI.
- Logging with `log/slog`; configuration from environment variables.

**Frontend**

- **React + TypeScript (strict) + Vite.** The PWA plugin for Vite handles the manifest and the
  service worker.
- **TanStack Query** for everything that comes from the server; React state only for what lives in
  the screen.
- **OpenAPI contract** (`backend/api/openapi.yaml`) as the single description of the API; TypeScript
  types are generated from it, and the development mocks (MSW) answer in the same shapes.
- **Tailwind CSS** for styling; `date-fns` with time-zone support for dates.
- **react-i18next** for Spanish and Catalan.

## Alternatives

- **GORM / Ent** — faster to start, but tenant filtering becomes implicit (hooks, scopes), and the
  generated SQL is harder to audit. sqlc keeps SQL as the source of truth.
- **chi / Echo / Gin** — fine libraries; the 1.22 mux removed the need, and the standard library is
  what every Go codebase shares.
- **Next.js** — server rendering buys nothing for an app behind a login, and would need a Node
  server in production. A static PWA is served by Caddy.
- **Native apps / React Native** — app-store accounts, reviews and two codebases for a product whose
  users only need to read a schedule. A PWA installs from the browser (N3).

## Consequences

- More SQL to write by hand; in exchange, every query is visible and reviewable.
- Generated code (sqlc, OpenAPI types) is committed, and CI fails if it is stale.
- Two toolchains in one repository (Go, Node); the Makefile is the shared entry point.
