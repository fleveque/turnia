package httpx

import (
	"log/slog"
	"net/http"
)

// Logger returns middleware that writes one log line per request, after the
// request has been handled. The line is at Info level, with the message
// "request" and four attributes:
//
//	method    r.Method, such as "GET"
//	path      r.URL.Path, such as "/api/v1/healthz"
//	status    the status code that was sent, such as 200
//	duration  how long handling the request took
//
// YOUR TURN (milestone 1). The outer layers are already written; only the
// inside of the handler below is left. Run the tests with
//
//	go test ./backend/internal/platform/httpx -run TestLogger -v
func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Four steps:
			//
			//  1. Note the time the request started. (time.Now)
			//  2. Make a statusRecorder that wraps w, so the status gets
			//     written down (see status.go). Make it with
			//     &statusRecorder{ResponseWriter: w}.
			//  3. Let the rest of the server handle the request, passing your
			//     recorder instead of w. (next.ServeHTTP)
			//  4. Log the line described above. The status is the recorder's
			//     Status(); the duration is time.Since(the start time).
			//
			// Then delete this line, which skips the logging:
			next.ServeHTTP(w, r)
		})
	}
}
