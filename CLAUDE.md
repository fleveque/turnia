# Turnia — notes for Claude Code

Shift scheduling for small pharmacies: a Go API on Postgres and a React PWA, multi-tenant (one
pharmacy = one tenant), Spanish and Catalan. Open source, AGPL-3.0. It doubles as the author's way of
learning Go and React: the commit history and `docs/lessons` are the learning record. Start with
[README.md](README.md) and [docs/design.md](docs/design.md).

## Status — 2026-10-09

- Milestone 0 (this PR): design, ADRs 0001–0008, compose Postgres on 5433, Makefile, lesson tooling,
  docs CI. No application code yet.

## Next

Milestone 1 — Go server skeleton (README roadmap). Then one milestone per PR, in roadmap order.

## Conventions

- **The design's non-negotiables (design §2, N1–N6) are not traded for convenience.** Tenant
  isolation is enforced by RLS (ADR 0002); shift changes are audited by a trigger (ADR 0005); every
  user-facing string exists in `es` and `ca`; shift type, weekend and "changed" are never conveyed by
  colour alone; employees see the whole team's shifts but never a colleague's hours, contract or
  absence details, and change shifts only through swaps (ADR 0007); Turnia staff are a separate
  account type that sees aggregates only, through functions `turnia_app` can't execute (ADR 0008).
- **One milestone = one PR = one idea**, about 150–400 lines of real code (generated code and
  lockfiles aside). Don't fold the next milestone into the current one.
- **Every milestone ships a lesson**: `docs/lessons/NN-*.md` (NN = milestone number), in the author's
  first person — someone who writes Elixir and Ruby daily, learned some Go in quantic-agent-go
  (modules, `net/http` client, errors, `context`, goose, SQLite) and is new to React. Build on that;
  don't re-teach it. Plus a formatted page published as an artifact, linked from the lesson and
  `docs/lessons/README.md`.
- **…and a code walkthrough** for every milestone with code: a second artifact going through every
  file the PR adds or changes, in reading order, as a tech lead mentoring someone new to the
  language: what each part does, why it's written that way, what the standard library or dependency
  does underneath, why each pointer is a pointer. Excerpts are copied verbatim from a named commit
  with `docs/lessons/pages/build.py`; "try it" boxes show real outputs, usually by breaking the code
  and showing the test that catches it. Compare with Elixir/Phoenix/LiveView or Ruby only where it
  genuinely helps.
- Every page uses the stylesheet in `docs/lessons/pages/` (carried over from quantic-agent). Reuse it,
  don't redesign it.
- **"Your turn"**: each PR description offers one small, bounded piece the author may write
  themselves (tests and skeleton provided). Default: Claude writes everything.
- **Decisions are ADRs** in `docs/decisions/NNNN-*.md`, indexed in its README.
- **`main` is protected.** Branch → PR → CI green → the author merges. Never merge. Before pushing to
  an existing branch, check its PR isn't already merged.
- **Verify CI on the exact commit SHA** (`gh run list --json headSha`), not the PR's check list.
- PRs are squash-merged, so a walkthrough built from a branch commit cites a SHA `main` won't have.
  After merge, rebuild the walkthrough from the merge commit and republish the same artifact.
- **Claims are verified by running them.** Lesson outputs are real outputs.
- Gate before every commit (grows with the code): docs relative-link check (`.github/workflows/ci.yml`);
  from milestone 1 `gofmt -l .` prints nothing, `go vet ./...`, `go test -race ./...`; from
  milestone 2 `sqlc diff`; from milestone 11 `npm run lint`, `typecheck`, `test`, `build`.

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
