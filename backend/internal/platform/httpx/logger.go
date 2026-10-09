package httpx

import (
	"log/slog"
	"net/http"
)

// Logger returns middleware that logs one line per request, once the
// handler has finished: at Info level, with the message "request" and these
// attributes:
//
//	method    the request method, such as "GET"
//	path      the request path, such as "/api/v1/healthz" (no query string)
//	status    the status code the handler sent; 200 if it never called
//	          WriteHeader (net/http sends 200 on the first Write)
//	duration  how long the handler took, as a time.Duration
//
// It must not change the response in any way: the client gets exactly the
// status, headers and body the handler wrote.
//
// YOUR TURN (milestone 1). The tests in logger_test.go describe what's
// expected; run them with
//
//	go test ./internal/platform/httpx -run TestLogger -v
//
// Hint: http.ResponseWriter has no method to ask which status was written.
// You'll need a small type of your own that wraps it and remembers.
func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return next // Replace this with a handler that calls next and then logs.
	}
}
