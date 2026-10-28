# Lesson 02 — SQL in, Go out

**Milestone 2**: Postgres. Migrations built into the binary, a database for every test, a
connection pool, a health check that asks the database, and sqlc, which turns hand-written SQL
into Go. I'd used goose and SQLite in quantic-agent-go; Postgres from Go, and generating code
from SQL, were new.

Coming from Elixir and Ruby, and from lesson 01.

*Also readable as a [formatted page](https://claude.ai/artifact/J4eoHobirDcgXNaLggHB3U). For the code itself,
file by file: walkthroughs [02.1 migrations](https://claude.ai/artifact/A7NYjfG1awRodmvib2oAT4),
[02.2 the pool](https://claude.ai/artifact/HEhGJDFdgxbCshy6ZGXUmM) and
[02.3 the queries](https://claude.ai/artifact/GTexHL9WKmrzQb4ZUnTgp1).*

---

## A comment the compiler reads

The migrations are SQL files, and a deployed `turnia` has no files next to it. They travel inside
the binary:

```go
//go:embed migrations/*.sql
var migrations embed.FS
```

It looks like a comment. It isn't: the compiler reads `//go:embed`, copies the matching files into
the program, and `migrations` becomes a small read-only file system that goose reads from. The
binary is all a deploy needs.

I wrote `turnia migrate` myself, as the 2.1 exercise: load the config, make the logger, call
`db.Migrate`, return 0 or 1. The same shape as `serve`, without the server.

## A database for every test

`go test` runs packages at the same time, so tests can't share one database.
`dbtest.NewDatabase(t)` creates one with a random name for each test and drops it when the test
ends, with `t.Cleanup`, which is ExUnit's `on_exit`. It's the Ecto SQL sandbox's job done the
blunt way: a whole database instead of a transaction rolled back. It costs a fraction of a second
per test and nothing can leak between them.

## A pool that hasn't connected yet

```go
pool, err := pgxpool.New(ctx, databaseURL)
...
if err := pool.Ping(ctx); err != nil {
	pool.Close()
	return nil, fmt.Errorf("connecting: %w", err)
}
```

`pgxpool.New` reads the URL and builds the pool, but doesn't connect. A wrong password or a
stopped database isn't noticed until something borrows a connection. So `db.Open` pings straight
away, and `serve` fails at start, where a deploy sees it, instead of on the first request.

The `pool.Close()` on failure surprised me. `New` has already started a goroutine that checks the
pool's connections every minute. The caller gets `nil` and an error, so nobody else could ever
close that pool, and every failed `Open` would leave a goroutine running.

The other difference from Ecto is where the pool lives. Ecto starts it under my supervision tree
and I reach it through the Repo module from anywhere. Here it's a value: `serve` makes it and
passes it to whatever needs it. Nothing is global.

## Asking for a method, not a type

The health check needs one thing from the database:

```go
type Pinger interface {
	Ping(ctx context.Context) error
}
```

`*pgxpool.Pool` fits it without saying so: it has the method, and that's all a Go interface asks.
Nothing says "implements". It's an Elixir behaviour without the `@behaviour`, or Ruby duck typing
checked by the compiler. Tests pass a three-line fake instead of a Postgres.

sqlc's generated code does the same, three methods wide:

```go
type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}
```

A pool, a single connection and a transaction all have those, so the same query runs inside a
transaction or outside one. Milestone 3 depends on that.

## My health check, and one missing line

The 2.2 exercise was the body of the health check:

```go
if err := db.Ping(ctx); err != nil {
	log.Warn("health check: database", "err", err)
	httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
	return
}

httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
```

The interesting line is `return`. Without it, the handler writes the 503 and then carries on to
the 200. Lesson 01's rule says the second status is ignored, so the client still sees 503, and a
monitor reading only the status would never notice. But the body is a stream, and it gets both:

```
--- FAIL: TestHealthzWithTheDatabaseDown (0.00s)
    server_test.go:50: body = "{\"status\":\"unavailable\"}\n{\"status\":\"ok\"}\n", want {"status":"unavailable"}
```

`Warn`, not `Error`, because when the database goes away the health check fails every few
seconds, and the error worth reading is whatever made it go away. The error goes to the log and
never to the response, because anyone can call that URL and the error can contain the database's
address.

## SQL in, Go out

The part of this milestone that looked strangest to me: I write SQL, and a tool writes the Go.

```sql
-- name: ListUsers :many
SELECT * FROM users
WHERE pharmacy_id = $1
ORDER BY name;
```

`go tool sqlc generate` reads that, checks it against the schema the migrations build, and writes
`func (q *Queries) ListUsers(ctx context.Context, pharmacyID uuid.UUID) ([]User, error)`, with the
`User` struct, the `Query`, the `Scan` of every column and the `rows.Err()` check at the end. My
first question was whether this is how it's usually done in Go. It's one of the usual ways:

| Approach | What you write | Examples |
|---|---|---|
| By hand | SQL strings, plus the `Scan` loop for every query | `database/sql`, or pgx directly |
| Helpers | SQL strings; struct tags map columns to fields | sqlx, pgx's `RowToStructByName` |
| Generated from SQL | `.sql` files; a tool writes the Go | sqlc, jet |
| ORM | Go structs and method chains; the library writes the SQL | GORM, ent, bun |

Writing it by hand is still common, and `users.sql.go` is exactly the code I'd have written: sqlc
writes the boring part and checks the column names on the way.

### Why generating code is normal in Go

Go has no macros and very little metaprogramming, and its generics are deliberately small. Where
another language would let the program write code for itself, Go has tools that write it to files:
`go generate`, `stringer` for constants' names, `protoc-gen-go` for gRPC, `mockgen` for mocks, and,
in milestone 13, TypeScript types from the OpenAPI contract. sqlc is one more.

Elixir does the same job at compile time with macros: `schema "users" do … end` writes the
`%User{}` struct and its functions, and I never see that code. Rails does it at run time:
ActiveRecord reads the columns when the app boots and defines the methods then. Go has neither,
so the code is a file I can open and read in a pull request.

### Why Turnia uses it

ADR 0001 chose sqlc over an ORM because tenant isolation is what this project can least afford to
get wrong. With sqlc, the SQL in a pull request is what Postgres runs. With GORM, filtering by
pharmacy tends to live in hooks or scopes, out of sight.

The weak spot is a query whose shape changes at run time, like a list with five optional
filters. sqlc wants one fixed query per function; the usual ways around it are optional
parameters in the SQL (`WHERE ($2::text IS NULL OR role = $2)`), several small queries, or a query
builder for that one place. I expect to meet it in the admin screens.

### What sqlc checks, and what it doesn't

A misspelled column stops it before any Go is written:

```
$ go tool sqlc generate
# package queries
internal/platform/db/queries/users.sql:7:1: column reference "nmae" not found: if you want to skip this validation, set 'strict_order_by' to false
```

But it isn't Postgres. `WHERE pharmacy_id = $1 AND active = 1`, a boolean compared with a number,
generates without a word, and only fails when it runs, in my test:

```
queries_test.go:104: ListUsers: ERROR: operator does not exist: boolean = integer (SQLSTATE 42883)
```

sqlc checks names. Behaviour is the tests' job.

## My first test

The 2.3 exercise was the other way round from the first two: Claude wrote the code, and I wrote
the test that specifies it. `ListUsers` should return one pharmacy's users, sorted by name. The
test makes two pharmacies and three users, calls the query, and compares the names with
`slices.Equal`, because `==` doesn't work on slices.

What makes it a good test is the data, not the code. Marta goes in before Jordi, so a query that
forgot to sort returns `[Marta Jordi]`. Pau is in the other pharmacy, so a query that forgot the
`WHERE` returns three names. Broken four ways, it catches each:

| `ListUsers` changed to | My test says |
|---|---|
| `WHERE pharmacy_id = $1 OR true` | `names = [Jordi Marta Pau], want [Jordi Marta]` |
| no `ORDER BY` | `names = [Marta Jordi], want [Jordi Marta]` |
| `ORDER BY name DESC` | `names = [Marta Jordi], want [Jordi Marta]` |
| `ORDER BY name LIMIT 1` | `names = [Jordi], want [Jordi Marta]` |

The review was about messages: Go's own tests say `names = %v, want %v`, got first, so every failure
reads the same way, and `"Something went wrong"` became `"ListUsers: %v"`, which says what failed.

## Tests that can't fail

This milestone gave me two more of these, after lesson 01's header test.

In 2.1, Claude wrote a test that ran two, then eight, migrations at once to prove goose's lock
keeps them apart. With the lock removed, it still passed: Postgres kept them apart by itself. A
test that passes either way proves nothing, so it was deleted. The lock stayed, because goose
recommends it.

In 2.3, I checked my test by changing an expected name, and it still passed. My test was fine;
it had never run:

```
$ go test -v -run ListUsers ./backend/internal/platform/db/queries
    queries_test.go:111: TURNIA_TEST_DATABASE_URL isn't set; `make test` sets it
--- SKIP: TestListUsersReturnsOnePharmacysUsersByName (0.00s)
PASS
ok
```

Without the variable, `dbtest` skipped every database test, and a skipped test counts as passed.
Without `-v`, all I saw was `ok`. Now it falls back to the development database and fails when
there isn't one.

So it's the same rule, three times: break the code, or the expectation, and watch the test go red.
Until I've seen it fail, I don't know it can.

## What I'm taking into milestone 3

- Files can travel inside the binary with `//go:embed`, a comment the compiler reads.
- A pool is a value I pass around, and `pgxpool.New` doesn't connect until something asks:
  ping at start.
- Interfaces are asked for by the code that uses them, one method if that's enough. A pool, a
  connection and a transaction are interchangeable to sqlc's code.
- After writing an error response, `return`: a second status is ignored, a second body isn't.
- In Go, the code a macro would write is generated into a file. sqlc checks names, not behaviour.
- A test isn't checked until I've watched it fail, and a skip says `ok`.
- Milestone 3 brings row-level security: every query will run in a transaction that has told
  Postgres which pharmacy it's for.
