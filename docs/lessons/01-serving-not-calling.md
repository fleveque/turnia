# Lesson 01 — Serving, not calling

**Milestone 1** — the Go server skeleton: config, a health check, error responses, panic recovery,
a request logger, graceful shutdown. In quantic-agent I used `net/http` as a client: build a
request, send it, read the answer. This milestone is the other side, and it turned out to have rules
the client side never showed me.

Coming from Elixir and Ruby, and from quantic-agent-go's milestones 0–8.

*Also readable as a [formatted page](https://claude.ai/artifact/3Ph6s31BFKVrwfV2V5A6ot). For the code itself, file
by file, see the [walkthrough](https://claude.ai/artifact/KSNkV2CncyoiwmY3FKYdVQ).*

---

## A handler is one method

The whole server side of `net/http` rests on one interface:

```go
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
```

A request comes in, `ServeHTTP` gets it, and writes the answer through the `ResponseWriter`. There's
no return value: the answer is *sent*, not returned. That's the first difference from Plug, where a
plug returns a new conn.

Routing is the standard `ServeMux`, which since Go 1.22 matches methods and path patterns:

```go
mux.HandleFunc("GET /api/v1/healthz", healthz)
mux.HandleFunc("/api/", notFound) // everything else under /api
```

### The 404 that should be a 405

A `POST` to `/api/v1/healthz` answered 404, not 405 Method Not Allowed. With the `/api/` catch-all
removed, the mux answers correctly by itself:

```
HTTP/1.1 405 Method Not Allowed
Allow: GET, HEAD
Content-Type: text/plain; charset=utf-8
```

The mux only answers 405 when *no* pattern matches the request, and `/api/` matches any method, so
it wins. The price of the catch-all is a wrong status for a wrong method; the price of removing it
is plain-text errors. I kept JSON everywhere. (Also: a `GET` pattern serves `HEAD` too.)

## Functions that return functions

Every middleware has the same type: give it a handler, get back a handler that wraps it.

```go
func Recover(log *slog.Logger) func(http.Handler) http.Handler
```

Three nested functions: one that takes the middleware's settings (a logger), one that takes the
next handler, and the handler itself, which does its work around `next.ServeHTTP(w, r)`. The server
stacks them like an onion:

```go
return httpx.Logger(log)(httpx.Recover(log)(mux))
```

A request goes in through Logger, then Recover, then the mux; the answer comes back out the other
way. Logger is on the outside so that it sees the 500 Recover writes after a panic.

What makes it shorter than it looks is `http.HandlerFunc(func(w, r) {…})`. That isn't a call: it
converts a plain function into a type that has a `ServeHTTP` method, so the function counts as a
`Handler`. The closest thing in Elixir is that a Plug can be a function or a module; here the
conversion is explicit.

## Headers, then status, then body

The rule the client side never taught me: a response goes out in a fixed order. The first
`WriteHeader` — or the first `Write`, which implies `WriteHeader(200)` — sends the status line and
the headers. After that, a header you set is silently dropped, and a second status is ignored.

That's why `httpx.JSON` encodes into a buffer before sending anything (a failure can still become a
clean 500), and why my logger needed help (below).

### The test that couldn't fail

To show the rule, I moved `Error`'s `Content-Type` after `WriteHeader`. The test still passed. It
read `rec.Header()` from `httptest.ResponseRecorder`, which is the live map handlers write into, so
it showed the header even though a real client would never have received it. The tests now read
`rec.Result().Header`, the snapshot taken when the status was sent:

```
--- FAIL: TestError (0.00s)
    respond_test.go:54: Content-Type = ""
```

Breaking the code on purpose is the only way I know to check that a test can fail.

## My first piece of Go in Turnia

The milestone's exercise was the request logger, and its first version was too much for me: it
needed both the response rules above and wrapping a `ResponseWriter` in a type of my own, at once,
explained in words I didn't have yet. We shrank it. Claude wrote `statusRecorder`, the small type
that notes which status was sent, and I wrote the handler body:

```go
start := time.Now()
rec := &statusRecorder{ResponseWriter: w}
next.ServeHTTP(rec, r)
log.Info("request",
	"method", r.Method,
	"path", r.URL.Path,
	"status", rec.Status(),
	"duration", time.Since(start),
)
```

It worked the first time I ran the tests. The review was about names, not behaviour:

- I'd called the start time `t`. In Go, `t` is the `*testing.T` in every test, so it reads wrong.
- I'd called the recorder `statusRecorder`, the same as its type. That compiles, but from that line
  on the variable hides the type inside the function:

  ```
  logger.go:23:14: statusRecorder (local variable) is not a type
  ```

  In Ruby a local variable and a class can't collide (`Foo` vs `foo`); in Go both are lowercase
  identifiers in the same scope, and the inner one wins.

Go likes short names for things that live three lines (`rec`, `start`) and longer ones as the scope
grows. That's the opposite of my Ruby habits.

And the server running with it:

```
level=INFO msg=request method=GET path=/api/v1/healthz status=200 duration=131.089µs
level=INFO msg=request method=GET path=/api/v1/shifts status=404 duration=46.448µs
level=INFO msg=request method=POST path=/api/v1/healthz status=404 duration=20.148µs
```

## Named parameters that aren't

What surprised me most was the call to `log.Info`. It looks like named parameters. Go doesn't have
any. The signature is:

```go
func (l *Logger) Info(msg string, args ...any)
```

`...any` is "any number of arguments, of any type", collected into one slice; slog reads it two at
a time, a name and a value. The compiler knows nothing about the pairing, so a mistake compiles and
logs nonsense:

```go
slog.Info("request", "method", "GET", "status")       // a name with no value
slog.Info("request", "method", "GET", status, "path") // a value where a name should be
```

```
INFO request method=GET !BADKEY=status
INFO request method=GET !BADKEY=404 !BADKEY=path
```

`go vet` catches both, and it runs in `make check` and CI:

```
main.go:7:2: call to slog.Info missing a final value
main.go:8:40: slog.Info arg "status" should be a string or a slog.Attr (possible missing key or value)
```

The stricter spelling is one typed value per pair, `slog.String("method", r.Method)`,
`slog.Int("status", 200)`, which can't get out of step.

Where Elixir has keyword lists and Ruby has keyword arguments, Go uses **structs** when a function
needs many named settings. I'd already used two without noticing: `&http.Server{ReadHeaderTimeout:
5 * time.Second, …}` and `&slog.HandlerOptions{Level: …}`. Named fields, checked by the compiler.
slog's pairs are the exception, chosen to keep log calls short.

## Stopping politely

When Kamal replaces the container, Docker sends `SIGTERM`. `signal.NotifyContext` turns that (or
Ctrl-C) into a cancelled `context.Context`, the same kind I used for deadlines in quantic-agent, and
`server.Run` reacts to it: stop accepting connections, give requests in flight up to ten seconds,
then return. A test proves it with a handler that takes 200ms and a stop in the middle: the request
still gets its answer.

The detail I'd have missed: the server runs in a goroutine that reports how it ended on a channel
created with `make(chan error, 1)`. The `1` is a buffer. If `Run` returns early and nobody ever reads
the channel, the goroutine's send still completes instead of blocking forever.

## The module isn't at the root

My first test run failed before running anything:

```
go: cannot find main module, but found .git/config in /home/fleveque/Projects/turnia
```

The module is in `backend/`, and Go looks for `go.mod` in the current directory and its parents.
A one-line `go.work` at the root (`use ./backend`) fixes it, for the terminal and for the editor.
Package paths stay relative to where you are: `./backend/internal/...` from the root.

## What I'm taking into milestone 2

- A handler sends its answer; it doesn't return it. Headers, then status, then body, and nothing
  changes after the status is sent.
- Middleware is `func(http.Handler) http.Handler`, stacked outside in.
- Test what the client receives, and break the code to check the test can fail.
- Go has no named parameters: structs for settings, and slog's name/value pairs checked by `go vet`.
- Short names for short scopes, and never a variable named like its type.
- Milestone 2 brings Postgres: a connection pool, migrations, and SQL turned into typed Go.
