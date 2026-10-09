# Lesson 00 — Decisions before code

**Milestone 0** — repository, design, decisions. No Go or React yet. What this milestone taught me is
that the two rules I care most about — *a pharmacy never sees another's data* and *no shift changes
without a record* — shouldn't live in my Go code at all. They belong in Postgres, and I tried them
there before writing a line of the application.

Coming from Elixir and Ruby, where Ecto and ActiveRecord made the database feel like storage with a
nice API on top.

*Also readable as a [formatted page](https://claude.ai/artifact/SAfdpXAmgvfTBSPWkJ3ec4).*

---

## Write the non-negotiables down first

In quantic-agent the design started with a short list of things the agent must never do, and every
later decision was checked against it. Turnia gets the same treatment ([design §2](../design.md#2-non-negotiables)):
N1 to N6. Two of them shape everything:

- **N1. A pharmacy never sees another pharmacy's data.**
- **N2. No shift changes without a record.**

The interesting question isn't *what* they say but *where they are enforced*. If N1 lives in a
`WHERE pharmacy_id = $1` that every query has to remember, it holds exactly as long as I never forget
one. In Rails I'd reach for a default scope and `acts_as_tenant`; in Phoenix, a `prefix` per tenant or
a `where` added in a context function. Both are the application promising to behave.

What I wanted instead is for the database to refuse. So before choosing anything else, I tried it.

## Row-level security, tried by hand

Postgres can attach a policy to a table: a condition every row must meet to be visible to a query.
In a scratch database, a `shifts` table with two pharmacies' rows, and this:

```sql
ALTER TABLE shifts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant ON shifts
  USING (pharmacy_id = current_setting('app.pharmacy_id')::int);
GRANT SELECT, INSERT ON shifts TO app;
```

`current_setting('app.pharmacy_id')` reads a variable that the connection sets for itself. Connected
as the `app` role, inside a transaction, I say which pharmacy I am:

```
BEGIN
 set_config
------------
 1

 person |       starts_at
--------+------------------------
 Laia   | 2026-10-12 07:00:00+00
 Marc   | 2026-10-12 12:00:00+00
(2 rows)
```

`SELECT person, starts_at FROM shifts` — no `WHERE` at all — and Núria, who works at pharmacy 2, isn't
there. The query didn't filter; the table did.

### It fails closed

The part that convinced me. If I forget to say who I am:

```
ERROR:  unrecognized configuration parameter "app.pharmacy_id"
```

An error, not every row. And the third argument of `set_config(…, true)` makes the setting
**local to the transaction**. On the same connection, after `COMMIT`:

```
ERROR:  invalid input syntax for type integer: ""
```

That matters because the Go server will use a connection pool. A connection that served pharmacy 2's
request goes back to the pool and is handed to pharmacy 1's next. If the setting outlived the
transaction, the second request would run as the first pharmacy — unless I remembered to reset it.
Transaction-local means there is nothing to remember. That's why every tenant query in Turnia will
run inside `db.WithTenant(ctx, pharmacyID, fn)` (milestone 3,
[ADR 0002](../decisions/0002-shared-schema-with-row-level-security.md)).

### The gotcha

The same `SELECT`, connected as `turnia` — the user the Docker image creates:

```
 pharmacy_id | person
-------------+--------
           1 | Laia
           1 | Marc
           2 | Núria
           1 | Laia
(4 rows)
```

Everything. `turnia` is a superuser, and **superusers bypass RLS**, always. The table owner bypasses
it too unless the table says `FORCE ROW LEVEL SECURITY`. So the app will connect as its own role,
`turnia_app`, which owns nothing and can bypass nothing; migrations run as the owner, from a different
connection string. And the isolation tests must connect as `turnia_app`, or they test nothing.

## One person, one place at a time

The fourth row above is the second experiment. An employee can't work two overlapping shifts. In
Rails that's a custom validation that loads the person's other shifts and compares — and two admins
saving at the same moment both pass it. Postgres has a constraint for this:

```sql
EXCLUDE USING gist (person WITH =, tstzrange(starts_at, ends_at) WITH &&)
```

*No two rows may have the same person and overlapping time ranges.* Laia works 9:00–14:00; giving her
13:00–17:00:

```
ERROR:  conflicting key value violates exclusion constraint "shifts_person_tstzrange_excl"
DETAIL:  Key (person, tstzrange(starts_at, ends_at))=(Laia, ["2026-10-12 11:00:00+00","2026-10-12 15:00:00+00")) conflicts with existing key (person, tstzrange(starts_at, ends_at))=(Laia, ["2026-10-12 07:00:00+00","2026-10-12 12:00:00+00")).
```

And 14:00–17:00, starting exactly when the morning ends, is accepted: `INSERT 0 1`. A `tstzrange` is
half-open by default — `[start, end)` — so back-to-back shifts don't overlap. That's the fourth row:
Laia's afternoon.

Ecto knows about this, as it happens: `exclusion_constraint/3` in a changeset turns the error into a
form error. In Go, pgx will hand me the Postgres error code (`23P01`) and the shift package will turn
it into a `409 Conflict`. Same idea, without the changeset.

## The audit log is the same argument

N2, "no shift changes without a record", follows the same reasoning. If the Go service writes the
audit row, then a new endpoint, a one-off script, or a fix typed into `psql` can change a shift
silently. A trigger on `shifts` can't be skipped by anything that writes to the table, and if the
transaction doesn't say who is acting, the trigger refuses ([ADR 0005](../decisions/0005-audit-shifts-with-a-database-trigger.md)).

The cost is real: behaviour that lives in SQL is invisible when reading Go. The answer is to say so
where it matters (the shift package's doc comment) and to have tests that try to cheat.

## Small decisions, also written down

- **The Go module lives in `backend/`, not at the repository root**: `github.com/fleveque/turnia/backend`.
  quantic-agent taught me the module path is the import prefix for every package; here it simply
  gets one more segment. The frontend has its own `package.json` beside it. One repository, two
  toolchains, and a `Makefile` at the root as the single entry point — the role `mix` aliases or a
  `Rakefile` would play, for both halves.
- **Postgres on port 5433**, because my machine already has one on 5432. And Postgres 18's Docker
  image moved its data directory one level down: the volume mounts `/var/lib/postgresql`. Both are
  in `CLAUDE.md` under "learned the hard way".
- **AGPL-3.0**, not MIT like quantic-agent ([ADR 0006](../decisions/0006-agpl-3.0.md)). quantic-agent
  is a tool; Turnia may become a hosted service, and the AGPL is the open-source licence that says
  "if you host a modified version, share the changes".
- **One origin for the app and the API** ([ADR 0003](../decisions/0003-auth-tokens-on-one-origin.md)),
  which is why the frontend won't be on Vercel: iOS Safari blocks cross-site cookies, and most
  pharmacy employees will open Turnia on an iPhone.
- **Deployed like quantic.finance** ([ADR 0004](../decisions/0004-kamal-on-the-shared-vps.md)): Kamal,
  same server, merging a PR deploys it. The Go binary embeds the built PWA, so one container serves
  both. Deploys start at milestone 4, so every milestone after that is live.

## What I'm taking into milestone 1

- The database enforces N1 and N2; Go code is the first lock, Postgres is the one that can't be
  forgotten.
- RLS fails closed, *if* the app connects as a role that can't bypass it. Superusers and owners can.
- Settings that scope a request are transaction-local, because the connections are pooled.
- `tstzrange` is `[start, end)`: back-to-back shifts are fine, overlaps are errors with a code.
- Milestone 1 is the server side of what quantic-agent did as a client: a `net/http` server, its
  handlers and middleware, and shutting it down cleanly.
