# 0004 — Everything on one Hetzner VM with Docker Compose

**Status:** accepted · **Date:** 2026-10-09

## Context

The first users are a few pharmacies in Spain. Traffic is tiny, and data is personal data about
employees (GDPR). The project is run by one person, so operations must be boring.

## Decision

One Hetzner Cloud VM in Germany (EU data residency), running Docker Compose with:

- **Caddy** — automatic HTTPS; serves the PWA's static files and proxies `/api/*` to the API (one
  origin, ADR 0003). Hashed assets cached forever; `index.html`, the manifest and the service
  worker never cached.
- **api** — the `turnia` binary in a distroless image; a one-shot `migrate` service runs first.
- **Postgres 18** — on a volume, with a nightly `pg_dump` shipped off the machine.

Images are built in CI and pushed to GitHub's container registry; deploying is `docker compose pull
&& docker compose up -d`.

## Alternatives

- **Vercel / Cloudflare Pages for the frontend** — a global CDN helps little for users in one
  region, and a second origin brings back CORS and cross-site cookies.
- **Fly.io** — pleasant, but more expensive at this scale and less predictable; the same image runs
  there if needed.
- **Managed Postgres** — better backups and failover, at several times the cost. The trigger to move
  is the first paying customer.

## Consequences

- A single point of failure, accepted for now; backups are tested by restoring them.
- Moving the database out, or putting a CDN in front, changes configuration only, not code.
