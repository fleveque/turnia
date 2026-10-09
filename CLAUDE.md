# Turnia — notes for Claude Code

Shift scheduling for small pharmacies: a Go API on Postgres and a React PWA, multi-tenant (one
pharmacy = one tenant), Spanish and Catalan. Open source, AGPL-3.0. It doubles as the author's way of
learning Go and React: the commit history and `docs/lessons` are the learning record. Start with
[README.md](README.md) and [docs/design.md](docs/design.md).

## Status — 2026-10-09

- Milestone 0 merged (#1): design, ADRs 0001–0008 (0004: Kamal to the shared VPS), compose
  Postgres on 5433, Makefile, lesson tooling, docs CI.
- Milestone 1 (in progress): Go server skeleton — `internal/config`, `internal/platform/httpx`
  (JSON, RFC 9457 problems, Recover), `internal/server` (routes, timeouts, graceful `Run`),
  `cmd/turnia serve`. **The author is writing `httpx.Logger`** (their first "your turn"); its tests
  fail on purpose until then. Lesson 01 and walkthrough 01 are written after their code lands.

## Next

Review the author's `httpx.Logger`, then lesson + walkthrough 01. Milestone 2 — Postgres (README roadmap). Then one milestone per PR, in roadmap order.

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
- **"Your turn"**: a PR may offer one small, bounded piece the author writes themselves: a stub with
  its doc comment, plus the tests that specify it, checked beforehand against a reference
  implementation kept out of the repo. CI is red until they push it; Claude reviews it on the PR,
  then the walkthrough and lesson use their code. Ask before each milestone which piece, if any,
  they want; the author took `httpx.Logger` in milestone 1.
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
- `postgres:18` images keep data in a versioned subdirectory: mount the volume at
  `/var/lib/postgresql`, not `/var/lib/postgresql/data`.
- The compose `turnia` user is a superuser, and superusers bypass RLS even with `FORCE`. RLS tests
  must connect as the non-owner app role.
- `current_setting('app.x')` errors if the setting was never defined in the session, and returns
  `''` once a transaction-local `set_config` has ended — both fail closed with an `::uuid` cast.
- A non-deferrable exclusion constraint is checked row by row, so exchanging the assignees of two
  overlapping shifts fails even in a single `UPDATE`. Declared `DEFERRABLE INITIALLY IMMEDIATE`
  and deferred in the swap transaction, it passes, and still rejects a swap that double-books at
  commit (tried 2026-10-09).
- Since Postgres 15, `public` grants no `CREATE` to ordinary roles: the migration owner needs
  `GRANT CREATE, USAGE ON SCHEMA public` (or owns the schema).
- New functions are executable by `PUBLIC` by default. Every `SECURITY DEFINER` function needs
  `REVOKE EXECUTE … FROM PUBLIC` and an explicit grant, or every role can call it.
- A table owner with `FORCE ROW LEVEL SECURITY` is filtered too: a `SECURITY DEFINER` function owned
  by it sees nothing without a tenant. Cross-tenant functions are owned by a `BYPASSRLS` role.
- A role with no grant on an RLS table may see the policy's cast error (`""` for integer) instead of
  `permission denied`: the planner evaluates the policy expression first. Either way it's refused;
  tests should assert refusal, not a specific message.
