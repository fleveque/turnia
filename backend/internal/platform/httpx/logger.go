package httpx

import (
	"log/slog"
	"net/http"
	"time"
)

// Logger returns middleware that writes one log line per request, after the
// request has been handled. The line is at Info level, with the message
// "request" and four attributes:
//
//	method    r.Method, such as "GET"
//	path      r.URL.Path, such as "/api/v1/healthz"
//	status    the status code that was sent, such as 200
//	duration  how long handling the request took
func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t := time.Now()
			statusRecorder := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(statusRecorder, r)
			log.Info("request", "method", r.Method, "path", r.URL.Path, "status", statusRecorder.Status(), "duration", time.Since(t))
		})
	}
}
