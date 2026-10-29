# Turnia — notes for Claude Code

Shift scheduling for small pharmacies: a Go API on Postgres and a React PWA, multi-tenant (one
pharmacy = one tenant), Spanish and Catalan. Open source, AGPL-3.0. It doubles as the author's way of
learning Go and React: the commit history and `docs/lessons` are the learning record. Start with
[README.md](README.md) and [docs/design.md](docs/design.md).

## Status — 2026-10-29

- Milestone 0 merged (#1): design, ADRs 0001–0008 (0004: Kamal to the shared VPS), compose
  Postgres on 5433, Makefile, lesson tooling, docs CI.
- Milestone 1 merged (#2): Go server skeleton — `internal/config`, `internal/platform/httpx` (JSON,
  RFC 9457 problems, Recover, `statusRecorder`, and `Logger`, written by the author),
  `internal/server` (routes, timeouts, graceful `Run`), `cmd/turnia serve`, `go.work` at the root.
  Lesson 01 and walkthrough 01 published.
- Milestone 2.1 merged (#4): goose migrations embedded in the binary (`db.Migrate`), migration
  00001 (pharmacies, users), `dbtest.NewDatabase` (a database per test), `turnia migrate` (written
  by the author). Walkthrough 02.1 published.
- Milestone 2.2 merged (#5): `db.Open` (pgxpool, pings at start), `serve` fails fast without a
  database, the health check takes a `server.Pinger` (its body written by the author).
  Walkthrough 02.2 rebuilt from the merge commit.
- Milestone 2.3 merged (#6): sqlc as a `go tool`, package `queries` (`CreatePharmacy`,
  `CreateUser`, `ListUsers`), uuid/time overrides, `make generate`, `sqlc diff` in CI; the
  `ListUsers` test written by the author. `dbtest` falls back to the dev database and fails
  instead of skipping. Walkthrough 02.3 rebuilt from the merge commit.
- Lesson 02 merged (#7): "SQL in, Go out".
- Milestone 3.1 merged (#8): role `turnia_app` (created by
  `dev/postgres-init/02-app-role.sql`, rerun by `make db-up` and CI, since roles are cluster-wide),
  migration 00002 grants it row access only, `serve` connects as it and `migrate` as the owner
  (`TURNIA_MIGRATE_DATABASE_URL`, written by the author), `dbtest.AppURL`. Walkthrough 03.1
  rebuilt from the merge commit.
- Milestone 3.2 merged (#9): migration 00003, RLS with `FORCE` on `users` and `pharmacies`,
  policy `tenant_isolation` on `current_setting('app.pharmacy_id')::uuid` (the `pharmacies` one
  written by the author). `tenancy_test.go` (`tenants`, `inTenant`) tests as `turnia_app`; no
  test can catch a missing `FORCE` while the owner is a superuser. Walkthrough 03.2 rebuilt from
  the merge commit.
- Milestone 3.3 merged (#10): `db.WithTenant(ctx, pool, pharmacyID, fn func(pgx.Tx) error)`
  (its body written by the author); `CreatePharmacy` takes its id (`uuid.NewV7()` in Go).
  `closePool` in the tests reports a connection still in use instead of hanging in
  `pgxpool.Pool.Close`. Walkthrough 03.3 rebuilt from the merge commit.
- Lesson 03 merged (#11): "One pharmacy at a time".
- Milestone 4.1 open (`milestone-4/image`): `Dockerfile` (cross-compiled static binary on
  distroless `static-debian12:nonroot`, commit passed as `REVISION` and set with `-ldflags -X
  main.revision`), `.dockerignore`, `make image`, CI job `image` (linux/arm64, not pushed).
  `turnia version` (its `version` function the author's exercise); `serve` logs the version.

## Next

Milestone 4 — deploy, planned with the author (2026-10-29). Hosts: `turnia.quantic.es` (Spanish)
and `turnia.quantic.cat` (Catalan), both on the shared proxy; choosing the language from the host
for signed-out pages belongs to milestones 5 and 14.

- 4.1 `milestone-4/image` — open, above. Exercise: `version`, from Go's build info.
- 4.2 `milestone-4/prod-roles` — the Postgres accessory's init script creates a non-superuser
  owner and `turnia_app`, passwords from secrets. A test migrates as a non-superuser owner and
  shows `FORCE` filtering it, the test 3.2 couldn't have. Exercise: that test's assertions.
- 4.3 `milestone-4/kamal` — `config/deploy.yml`, `.kamal/secrets` (Bitwarden item `turnia`),
  `docs/deploy.md` with quantic's safety rules, proxy health check on `/api/v1/healthz`.
  **Migrations run before the new container boots, in a one-off container that holds only the
  owner URL** (the author's choice (a), 2026-10-29, keeping 3.1's rule that the server holds only
  the weaker URL); start with a spike on how a Kamal 2 pre-deploy hook gets that secret, and if
  it's clumsy, fall back to migrating on boot and say so. Amend ADR 0004 either way. The author
  runs the first `kamal setup` (their shared server; Claude doesn't run commands against it).
- 4.4 `milestone-4/deploy-workflow` — `deploy.yml` adapted from quantic's (bot refusal,
  Bitwarden, deploy key, SSH alias, blocking pre-deploy `pg_dump`, `[skip-snapshot]` in the
  merge title, native ARM runner). No code exercise; the author sets the repo secrets.
- 4.5 `milestone-4/backups` — Turnia's own `ops/backup/` (restic to the NAS, append-only, its
  own repository and HTTP user, monthly restore drill), `docs/backup.md`. The author sets up the
  NAS user and runs the first drill.
- Lesson 04 as its own PR. Still deferred: login's `SECURITY DEFINER` email lookup (milestone 5),
  `turnia_platform` (milestone 12).

## Conventions

- **The design's non-negotiables (design §2, N1–N6) are not traded for convenience.** Tenant
  isolation is enforced by RLS (ADR 0002); shift changes are audited by a trigger (ADR 0005); every
  user-facing string exists in `es` and `ca`; shift type, weekend and "changed" are never conveyed by
  colour alone; employees see the whole team's shifts but never a colleague's hours, contract or
  absence details, and change shifts only through swaps (ADR 0007); Turnia staff are a separate
  account type that sees aggregates only, through functions `turnia_app` can't execute (ADR 0008).
- **No passwords** (ADR 0003): sign-in is an email with a 6-digit code *and* a link, because an
  installed iOS PWA doesn't share cookies with Safari, where Mail opens links. Invite-only. Google
  sign-in comes after the MVP and links to existing users by verified email.
- **Small PRs; a milestone is as many PRs as it needs.** Each PR is one concept the author can
  review and understand in one sitting: **about 250 lines of Go/TS/SQL including tests, ideally
  less**, and rarely more than five or six files (generated code, lockfiles and docs not counted).
  Tests count because they're half of what gets read. Plan a milestone's PRs before starting it,
  name them `milestone-N/<topic>` (e.g. `milestone-2/pool`, `milestone-2/migrations`) and title them
  "Milestone N.k: …". Each PR builds, passes CI and is useful on its own, so it can be merged before
  the next one is opened. Milestone 1 (#2, ~830 lines) was too big and is the cautionary example: it
  should have been config + `turnia serve`, then `httpx`, then `server` with graceful shutdown.
- **Every milestone ships a lesson**, written with its last PR: `docs/lessons/NN-*.md` (NN = milestone number), in the author's
  first person — someone who writes Elixir and Ruby daily, learned some Go in quantic-agent-go
  (modules, `net/http` client, errors, `context`, goose, SQLite) and is new to React. Build on that;
  don't re-teach it. Plus a formatted page published as an artifact, linked from the lesson and
  `docs/lessons/README.md`.
- **…and every PR with code ships its walkthrough** with it, so the walkthrough is as short as the
  PR and is read during review: a second artifact going through every file the PR adds or changes, in reading order, as a tech lead mentoring someone new to the
  language: what each part does, why it's written that way, what the standard library or dependency
  does underneath, why each pointer is a pointer. Excerpts are copied verbatim from a named commit
  with `docs/lessons/pages/build.py`; "try it" boxes show real outputs, usually by breaking the code
  and showing the test that catches it. Compare with Elixir/Phoenix/LiveView or Ruby only where it
  genuinely helps.
- Every page uses the stylesheet in `docs/lessons/pages/` (carried over from quantic-agent). Reuse it,
  don't redesign it.
- **"Your turn" in every PR**, unless a PR has nothing worth learning by writing it (the author's
  choice, 2026-10-14). Each is one small, bounded piece the author writes themselves: a stub with
  its doc comment, plus the tests that specify it, checked beforehand against a reference
  implementation kept out of the repo. CI is red until they push it; Claude reviews it on the PR,
  then the walkthrough and lesson use their code. When planning a milestone, propose one exercise
  per PR and say why if a PR has none. The author took `httpx.Logger` in milestone 1.
- **Exercises fit a Go beginner.** The author found the first version of the logger exercise too
  hard: it needed two new concepts at once (how `http.ResponseWriter` sends a response, and
  wrapping an interface with embedding) plus jargon. It was shrunk: Claude wrote `statusRecorder`,
  the author writes the ~5-line handler body from four steps described in words. Rules: one new
  idea per exercise; prefer plain logic (functions, structs, slices, `time`) over framework
  machinery; scaffold the parts that need concepts not taught yet; explain in plain words and
  define every term used, in the stub's comments and in the walkthrough.
- **The same when TypeScript and React start (milestone 14).** The author is new to both. First
  exercises are plain TypeScript functions with Vitest tests: week dates from a day, minutes to
  "7h 30min", grouping shifts by day. No JSX, hooks or React concepts until they've been taught;
  then a component that only receives props and renders, before any state, effects or data
  fetching.
- **Decisions are ADRs** in `docs/decisions/NNNN-*.md`, indexed in its README.
- **`main` is protected** like quantic-agent's: PR required (0 approvals), admins included, linear
  history, conversations resolved, required checks. Branch → PR → CI green → the author merges.
  Never merge, never push to `main`. From milestone 4, a merge to `main` deploys. Before pushing to
  an existing branch, check its PR isn't already merged.
- **Verify CI on the exact commit SHA** (`gh run list --json headSha`), not the PR's check list.
- PRs are squash-merged, so a walkthrough built from a branch commit cites a SHA `main` won't have.
  After merge, rebuild the walkthrough from the merge commit and republish the same artifact.
- **Claims are verified by running them.** Lesson outputs are real outputs.
- Gate before every commit (grows with the code): docs relative-link check (`.github/workflows/ci.yml`);
  from milestone 1 `gofmt -l .` prints nothing, `go vet ./...`, `go test -race ./...`; from
  milestone 2 `sqlc diff`; from milestone 14 `npm run lint`, `typecheck`, `test`, `build`.

## Deploying (from milestone 4)

- Kamal to the VPS shared with quantic and older apps, following quantic's `docs/deploy.md`:
  **never `kamal proxy reboot`/`remove`** (the proxy is shared), never touch other apps' volumes,
  Kamal pinned to the ecosystem's version (2.7.0), Turnia's Postgres on `127.0.0.1:5435`.
- Secrets live in Bitwarden (item `turnia`); no server addresses or IPs in the repo.

## Learned the hard way

- Host port 5432 is taken by a local Postgres on the author's machine: dev Postgres is on **5433**.
- The Go module is in `backend/`, not the root. `go.work` at the root (`use ./backend`) lets `go`
  commands and gopls work from the root, with paths like `./backend/...`; package paths in
  `go test ./internal/...` are relative to the current directory, so those run from `backend/`.
- `postgres:18` images keep data in a versioned subdirectory: mount the volume at
  `/var/lib/postgresql`, not `/var/lib/postgresql/data`.
- The compose `turnia` user is a superuser, and superusers bypass RLS even with `FORCE`. RLS tests
  must connect as the non-owner app role.
- `current_setting('app.x')` errors if the setting was never defined in the session, and returns
  `''` once a transaction-local `set_config` has ended — both fail closed with an `::uuid` cast.
- sqlc's Postgres parser is C (pg_query_go), so `go tool sqlc` needs cgo and a C compiler, and
  its first build takes about 1.5 CPU-minutes. Later runs come from the build cache.
- goose's session locker (an advisory lock) is on in `db.Migrate`, but a test running 2 or 8
  migrations at once passed *without* it too (2026-10-14): Postgres serialised the DDL itself. So
  there's no test claiming the lock works; don't write one that can't fail.
- A non-deferrable exclusion constraint is checked row by row, so exchanging the assignees of two
  overlapping shifts fails even in a single `UPDATE`. Declared `DEFERRABLE INITIALLY IMMEDIATE`
  and deferred in the swap transaction, it passes, and still rejects a swap that double-books at
  commit (tried 2026-10-09).
- Since Postgres 15, `public` grants no `CREATE` to ordinary roles: the migration owner needs
  `GRANT CREATE, USAGE ON SCHEMA public` (or owns the schema).
- `pgxpool.Pool.Close` waits for every acquired connection, so a transaction never ended hangs a
  test at cleanup, with no message. Check `pool.Stat().AcquiredConns()` first (`closePool`).
- New functions are executable by `PUBLIC` by default. Every `SECURITY DEFINER` function needs
  `REVOKE EXECUTE … FROM PUBLIC` and an explicit grant, or every role can call it.
- A table owner with `FORCE ROW LEVEL SECURITY` is filtered too: a `SECURITY DEFINER` function owned
  by it sees nothing without a tenant. Cross-tenant functions are owned by a `BYPASSRLS` role.
- A role with no grant on an RLS table may see the policy's cast error (`""` for integer) instead of
  `permission denied`: the planner evaluates the policy expression first. Either way it's refused;
  tests should assert refusal, not a specific message.
