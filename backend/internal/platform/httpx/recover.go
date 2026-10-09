package httpx

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover returns middleware that turns a panic in a handler into a 500
// Problem response and an error log with the stack trace, instead of a
// dropped connection.
//
// net/http already recovers panics so one bad request can't kill the
// server, but it does so by closing the connection: the client sees a
// network error, not a response, and the log has no request context.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				// http.ErrAbortHandler is net/http's way of saying "stop
				// here, on purpose"; re-panic so it keeps its meaning.
				if v == http.ErrAbortHandler {
					panic(v)
				}
				log.Error("panic in handler",
					"method", r.Method,
					"path", r.URL.Path,
					"panic", v,
					"stack", string(debug.Stack()),
				)
				Error(w, http.StatusInternalServerError, "")
			}()
			next.ServeHTTP(w, r)
		})
	}
}
