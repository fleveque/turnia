# 0004 — Deployed with Kamal to the shared VPS, on every merge to main

**Status:** accepted · **Date:** 2026-10-09 · **Supersedes:** an earlier draft (a dedicated VM with
Docker Compose and Caddy, deployed by hand)

## Context

The first users are a few pharmacies in Spain; traffic is tiny, and the data is personal data about
employees (GDPR, EU hosting). The project is run by one person, who already runs quantic.finance and
some older apps on one Hetzner VPS (ARM) with Kamal: merging a PR deploys it, with secrets in
Bitwarden, a pre-deploy database snapshot, and restic backups to a NAS with a monthly restore drill
(quantic's `docs/deploy.md` and `docs/backup.md`). Building a second, different way to deploy would
be more to learn and more to keep working, for no gain.

## Decision

**Same server, same tools, same release process as quantic.**

- **Kamal 2**, pinned to the version the shared kamal-proxy accepts (2.7.0 today, as quantic and
  pulse). The proxy routes by host name, terminates TLS (Let's Encrypt), and switches containers
  with no downtime. Turnia registers its own host and never restarts the proxy.
- **One image, one container.** The Docker build compiles the frontend (Vite) and then the Go binary,
  which embeds the built PWA (`embed.FS`) and serves it next to `/api`. The app and the API are one
  origin (ADR 0003) without a separate web server. The binary is static, so the final image is
  distroless and small. Built natively for arm64 on GitHub's ARM runners.
- **Postgres 18 as a Kamal accessory** (`turnia-db`), bound to `127.0.0.1:5435` (quantic's is on
  5434), with its own volume. The app connects over Kamal's Docker network with three roles: the
  migration owner, `turnia_app` and `turnia_platform` (ADRs 0002, 0008).
- **Migrations on boot**: the container runs `turnia migrate` (as the owner) before `turnia serve`.
  goose takes a Postgres advisory lock, so two containers booting during a deploy take turns.
- **Release = merge.** A `deploy` workflow runs when CI succeeds on `main`, adapted from quantic's:
  refuse a push to `main` by a bot; fetch secrets from Bitwarden (item `turnia`); SSH through a
  host alias with the address in a repository secret (no IPs in the repo) and a dedicated deploy
  key; take a blocking `pg_dump` snapshot on the server first (`[skip-snapshot]` in the merge
  title as break-glass); `kamal deploy`. `kamal deploy` from a laptop remains the manual path.
- **Backups**: the quantic restic setup, extended to Turnia's volume — nightly to the NAS,
  append-only, with the monthly automated restore drill.
- **Early, not last.** Deploying arrives at milestone 4, right after tenancy, to a beta subdomain
  of a domain already owned. From then on every merged milestone is live, and the pipeline is
  exercised two dozen times before the first pharmacy. The product domain is chosen later; moving
  hosts is a kamal-proxy re-route, not a migration.
- Email through Resend (ADR 0003) is the one external service.

## Alternatives

- **A dedicated VM with Docker Compose and Caddy** (the earlier draft) — full isolation, but a second
  deploy method, a second proxy, a second backup setup, and manual releases.
- **A new VPS with Kamal** — same tools, separate blast radius, a few euros a month more and one
  more server to patch and back up. Revisit when Turnia has paying customers or needs more
  resources than the shared server has.
- **Vercel / Cloudflare Pages for the frontend** — a CDN helps little for users in one region, and
  a second origin brings back CORS and cross-site cookies.
- **Managed Postgres** — better failover, several times the cost. The trigger to move is the first
  paying customer.

## Consequences

- Turnia inherits quantic's safety rules on that server: never `kamal proxy reboot` or `remove`,
  never touch another app's volumes, mind host ports. They are copied into Turnia's deploy docs.
- A noisy neighbour can slow Turnia down, and the server is a single point of failure for every
  app on it; accepted at this size, with backups tested by restoring them.
- Upgrading Kamal is an ecosystem-wide change, done for all apps together.
