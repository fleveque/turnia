# Lesson 03 — One pharmacy at a time

**Milestone 3**: tenancy. Every pharmacy's rows are in the same tables, and Postgres, not the
code, makes sure a query only ever sees one pharmacy's. A second database role, row-level
security, and one Go function every handler will go through. Row-level security was new to me;
so was ending a transaction by hand in Go.

Coming from Elixir and Ruby, and from lessons 01 and 02.

*Also readable as a [formatted page](https://claude.ai/artifact/PESSxwcCcUEz629pzCYAux). For the code itself,
file by file: walkthroughs [03.1 the app role](https://claude.ai/artifact/BTAuhJaDNSSejptavueBPQ),
[03.2 the policies](https://claude.ai/artifact/8HHxrvDYLnXuX5KhieNpeT) and
[03.3 WithTenant](https://claude.ai/artifact/FUzWjgLV1bu5CxP5CJNExo).*

---

## The WHERE nobody can forget

Until this milestone, keeping one pharmacy's employees away from another's was the job of
`WHERE pharmacy_id = $1`, in every query, written by me, forever. In Rails that's
`acts_as_tenant` or a `default_scope`; in Ecto, a `prepare_query` callback. They work until
someone writes `unscoped`, raw SQL, or a new query that skips the hook.

ADR 0002 moved the job into Postgres. **Row-level security** adds an invisible condition to every
statement on a table, from rules called policies. A query can narrow it but can't widen it. A
`SELECT * FROM users` with no `WHERE` at all sees one pharmacy:

```
=> SELECT name FROM users;
ERROR:  unrecognized configuration parameter "app.pharmacy_id"
=> BEGIN;
=> SELECT set_config('app.pharmacy_id', '00000000-0000-7000-8000-000000000001', true);
=> SELECT name FROM users;
 name
------
 Laia
 Pau
(2 rows)
```

Three users exist; Marta is in the other pharmacy. The first query is the part I like most, and
it comes later.

## Who's asking

The first surprise came before any policy: who the policies apply to. Superusers skip row-level
security entirely, and so, by default, does a table's owner. Until 3.1 the server connected as
`turnia`, the development superuser that owns every table. Isolation would have been switched on
and checked by nobody.

So the server now connects as `turnia_app`, a role that owns nothing. It can read and write rows,
table by table, through grants in the migrations, and it can't create, alter, truncate or drop
anything:

```
ERROR:  permission denied for schema public      ← CREATE TABLE notes (id int)
ERROR:  must be owner of table users             ← ALTER TABLE users ADD COLUMN notes text
ERROR:  permission denied for table users        ← TRUNCATE users
ERROR:  must be owner of table users             ← DROP TABLE users
```

A Rails app usually connects as the same user that runs `db:migrate`, so a stray `DROP TABLE` in a
request would work. Here it can't.

Roles belong to the whole Postgres server, not to one database, so creating `turnia_app` isn't a
migration: migrations run once per database, and the tests create a database each. It's a script
that `make db-up` and CI run every time, safe to run twice because PL/pgSQL can catch
`duplicate_object`. Postgres has `CREATE TABLE IF NOT EXISTS`, but no `CREATE ROLE IF NOT EXISTS`.

### Two URLs, never one

The 3.1 exercise was mine: `MigrateDatabaseURL` in `config.Load`, a default and an environment
variable, the same shape as the five settings already there. The interesting part was a line I
didn't write. A test checks that each URL has its own default, and that setting one doesn't change
the other. The tempting shortcut, falling back to the server's URL when the migrate one isn't set,
fails it:

```
--- FAIL: TestLoadKeepsTheTwoDatabaseURLsApart (0.00s)
    config_test.go:68: MigrateDatabaseURL = "postgres://app@prod/turnia", want the default "postgres://turnia:turnia@localhost:5433/turnia?sslmode=disable"
```

In production that's a deploy that set only the server's URL by mistake, and migrations running
as the role that can't migrate, failing halfway with `permission denied`. Two settings, two
secrets, and the server's container only ever holds the weaker one.

## An error, not an empty page

The policy on `users` is two lines:

```sql
CREATE POLICY tenant_isolation ON users
    USING (pharmacy_id = current_setting('app.pharmacy_id')::uuid);
```

`current_setting` reads a setting, a named value that lives on the database connection, like
`timezone`. The server sets `app.pharmacy_id` at the start of each transaction, and the policy
compares it with every row.

What the policy *doesn't* say mattered most. `current_setting(name, true)`, "missing ok", returns
`NULL` for a setting nobody set, and `pharmacy_id = NULL` is never true. A forgotten tenant would
return zero rows: a page that says "no shifts this week", which looks like data, not like a bug.
Without that `true`, the same mistake is the error at the top of this lesson, the first time the
code runs.

There's a second way to have no tenant, and it fails too. Once a transaction-local setting has
ended, Postgres remembers the name and reads it as `''`, and `''::uuid` isn't a UUID:

```
=> ROLLBACK;
=> SELECT name FROM users;
ERROR:  invalid input syntax for type uuid: ""
```

I hadn't expected row-level security to be this strict by default either. A table with RLS on and
no policy shows nothing to anyone it applies to.

### My policy, and a Down that matters

The 3.2 exercise was the policy on `pharmacies`. It had RLS on and no policy, so `turnia_app` saw
no pharmacy at all. The one new idea: a pharmacy row doesn't *have* a pharmacy id, it *is* one,
so the condition compares the setting with `id`:

```sql
CREATE POLICY tenant_isolation ON pharmacies
    USING (id = current_setting('app.pharmacy_id')::uuid);
```

Before I wrote it, one failure surprised me: `SELECT * FROM pharmacies with no tenant: err = nil,
want an error`. No policy means no condition, so nothing ever calls `current_setting`, and nothing
errors. You just get nothing.

I also dropped the policy in the migration's Down section. A policy survives RLS being turned off,
so without that line the next `goose up` would fail with "policy already exists". Down then Up on
a migrated database went from 2 policies to 0 and back to 2.

### Invisible or refused

With only `USING`, the same condition decides what a statement can see and what it can write.
That gives two different failures, and I had to see both to believe them:

```
=> UPDATE users SET locale = 'ca' WHERE name = 'Marta';
UPDATE 0
=> INSERT INTO users (pharmacy_id, …) VALUES ('…0002', …);
ERROR:  new row violates row-level security policy for table "users"
```

Marta is in the other pharmacy, so the `UPDATE` can't see her and matches nothing, with no error.
Writing a row *into* the other pharmacy is refused outright. The tests have both lists, and a list
of writes that must still *work*: a policy of `USING (false)` would pass every refusal.

### What a policy doesn't hide

Constraints see every row. `users.email` is unique across all pharmacies, so adding a user with
Marta's email, from the other pharmacy, fails with `duplicate key value violates unique constraint
"users_email_key"`. That tells you the email exists somewhere. It's harmless for an invite, and
it's why login's email lookup gets a function of its own in milestone 5.

## WithTenant, and ending a transaction

Everything above needs the setting set, inside a transaction, on every request. 3.3 is the one
function that does it:

```go
func WithTenant(ctx context.Context, pool *pgxpool.Pool, pharmacyID uuid.UUID, fn func(tx pgx.Tx) error) error
```

It's the shape of Ecto's `Repo.transaction(fn -> … end)`: the work goes in as a function, and the
transaction is handled around it. Go has no `Repo.rollback/1`: returning an error from `fn` is how
it asks for a rollback. The body was my exercise:

```go
tx, err := pool.Begin(ctx)
if err != nil {
	return fmt.Errorf("beginning the transaction: %w", err)
}

defer tx.Rollback(ctx)

if _, err := tx.Exec(ctx, "SELECT set_config('app.pharmacy_id', $1, true)", pharmacyID.String()); err != nil {
	return fmt.Errorf("setting the tenant: %w", err)
}

if err := fn(tx); err != nil {
	return err
}

if err := tx.Commit(ctx); err != nil {
	return fmt.Errorf("committing: %w", err)
}

return nil
```

The new idea was `defer tx.Rollback(ctx)`, Go's `ensure`. It runs however the function returns,
and after a `Commit` it does nothing (pgx returns `ErrTxClosed`, which the `defer` throws away). So
one line covers every failure, and the success path doesn't have to undo it. It goes right after
`Begin` succeeds and not before: on an error `tx` is `nil`, and the deferred call would panic.

Two smaller things I'd have got wrong without the tests:

- **`set_config(…, true)`, not `SET LOCAL`.** `SET` can't take a `$1` parameter
  (`PREPARE p(text) AS SET LOCAL app.pharmacy_id = $1` is a syntax error), and a tenant id pasted
  into a SQL string is an injection waiting to happen. A function call takes parameters.
- **`fn`'s error comes back as it is.** Our own steps wrap theirs with `%w`, which adds context
  and keeps the original for `errors.Is` and `errors.As`. But `fn`'s error is the caller's, and
  wrapping it with `%v` turns it into text:

```
--- FAIL: TestWithTenantRollsBackAndReturnsFnsError (0.16s)
    tenant_test.go:78: err = fn: boom, want fn's error
```

### The connection that remembers

A pool reuses connections, and a setting lives on a connection. With `false` instead of `true`,
the setting lasts the whole session, and the next request on that connection runs as the previous
user's pharmacy. Every other test passes. One test catches it, with a pool of exactly one
connection so "the next request" is certainly the same connection:

```
--- FAIL: TestWithTenantLeavesNoTenantOnTheConnection (0.16s)
    tenant_test.go:115: app.pharmacy_id = "00000000-0000-7000-8000-000000000001" after WithTenant, want it empty
    tenant_test.go:118: a query after WithTenant, without it: err = nil, want an error
```

The second line is the worst bug this milestone could have: a query with no tenant, answered with
someone else's rows, silently.

## A test that hung

Checking the tests against wrong versions of `WithTenant`, Claude left out the
`defer tx.Rollback`. The tests didn't fail. They stopped:

```
$ go test -timeout 20s -run WithTenant ./internal/platform/db/
=== RUN   TestWithTenantRollsBackAndReturnsFnsError
panic: test timed out after 20s
github.com/jackc/pgx/v5/pgxpool.(*Pool).Close(0x0?)
```

The test's own checks had passed. It was stuck in the cleanup: `pgxpool.Pool.Close` waits until
every lent-out connection comes back, and the one held by a transaction nobody ended never would.
Without `-timeout`, `go test` waits ten minutes. A helper now asks the pool first,
`pool.Stat().AcquiredConns()`, and fails with a sentence instead:

```
--- FAIL: TestWithTenantRollsBackAndReturnsFnsError (0.26s)
    tenancy_test.go:50: 1 connections still in use at the end of the test: a transaction was never ended
```

In a server, the same bug would keep one connection per failed request until the pool was empty
and every request waited.

## Tests that can't fail, again

Two this time, and one we kept anyway.

`FORCE ROW LEVEL SECURITY` applies the policies to the table's owner too. In development and CI
the owner is a superuser, and superusers skip RLS even with `FORCE`. So no test can show it
working: delete the line and everything passes. It's in the migration because the design requires
it for production, where the owner is a normal role, and the walkthrough says plainly that nothing
tests it. A line nobody can test is fine; a test that pretends to is not.

The other one passed for the wrong reason. The "no tenant" test first checked `users` and
`pharmacies` in one loop, alternating "never set" and "set to empty". Breaking the policy with
"missing ok" was caught for `users` but not `pharmacies`: by then the pooled connection had already
seen the setting's name, so "never set" had become `''`, which fails on the cast either way. Two
loops, all the "never set" checks first, and both tables are caught. Lesson 02's rule again: watch
it fail, for each thing it claims.

## A migration runs once

goose records each migration's version and never runs it again. Had I run `make migrate` on the
stub of 00003, editing the file afterwards would have changed nothing on my database: version 3
recorded, no `pharmacies` policy. So I migrated only after my policy was in. The tests don't
care, because each one migrates a new database, which is exactly why they wouldn't have noticed.
Once a migration is on `main`, it's never edited; a change is a new migration.

## What I'm taking into milestone 4

- The role that runs the app shouldn't own the tables. Superusers, and owners without `FORCE`,
  skip row-level security.
- A policy is an invisible `WHERE`. Another tenant's rows are invisible to reads, updates and
  deletes, and writing into another tenant is refused.
- No tenant should be an error, not zero rows: `current_setting` without "missing ok".
- `defer tx.Rollback(ctx)` right after `Begin`, `Commit` at the end: the transaction ends on
  every path. Wrap my own errors with `%w`; return the caller's as they are.
- Settings live on connections, and pools reuse connections: transaction-local, always.
- A transaction left open doesn't fail a test, it hangs it. Ask the pool before closing it.
- Constraints see across tenants. And a line nobody can test is better than a test that can't fail.
- Milestone 4 deploys Turnia, with production roles and their passwords: the owner there won't be
  a superuser, and `FORCE` starts to matter.
