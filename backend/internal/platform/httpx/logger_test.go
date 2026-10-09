package httpx

// Tests for Logger, written before it: they are the specification.
// Your turn is to make them pass. Read them first, then logger.go.

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// serve runs one request through Logger(h) and returns the response and
// every log line, each decoded from JSON.
func serve(t *testing.T, h http.Handler, method, target string) (*httptest.ResponseRecorder, []map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	rec := httptest.NewRecorder()
	Logger(log)(h).ServeHTTP(rec, httptest.NewRequest(method, target, nil))

	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line isn't JSON: %q", line)
		}
		lines = append(lines, m)
	}
	return rec, lines
}

// only returns the single log line, failing if there isn't exactly one.
func only(t *testing.T, lines []map[string]any) map[string]any {
	t.Helper()
	if len(lines) != 1 {
		t.Fatalf("got %d log lines, want exactly 1: %v", len(lines), lines)
	}
	return lines[0]
}

func TestLoggerLogsMethodPathAndStatus(t *testing.T) {
	_, lines := serve(t, http.NotFoundHandler(), "POST", "/api/v1/nope?x=1")
	line := only(t, lines)

	if line["level"] != "INFO" || line["msg"] != "request" {
		t.Errorf("level/msg = %v/%v, want INFO/request", line["level"], line["msg"])
	}
	if line["method"] != "POST" {
		t.Errorf("method = %v, want POST", line["method"])
	}
	if line["path"] != "/api/v1/nope" {
		t.Errorf("path = %v, want /api/v1/nope (without the query string)", line["path"])
	}
	// JSON numbers decode into float64 when the target is `any`.
	if line["status"] != float64(404) {
		t.Errorf("status = %v, want 404", line["status"])
	}
}

func TestLoggerStatusIs200WhenTheHandlerOnlyWrites(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello")) // no WriteHeader: net/http sends 200
	})
	_, lines := serve(t, h, "GET", "/")

	if got := only(t, lines)["status"]; got != float64(200) {
		t.Errorf("status = %v, want 200", got)
	}
}

func TestLoggerStatusIs200WhenTheHandlerWritesNothing(t *testing.T) {
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	_, lines := serve(t, h, "GET", "/")

	if got := only(t, lines)["status"]; got != float64(200) {
		t.Errorf("status = %v, want 200", got)
	}
}

func TestLoggerLogsTheFirstStatusOnly(t *testing.T) {
	// Only the first WriteHeader counts; net/http ignores later ones (and
	// logs "superfluous response.WriteHeader call").
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.WriteHeader(http.StatusInternalServerError)
	})
	rec, lines := serve(t, h, "GET", "/")

	if got := only(t, lines)["status"]; got != float64(rec.Code) {
		t.Errorf("logged status %v, but the client got %d", got, rec.Code)
	}
}

func TestLoggerLogsTheDuration(t *testing.T) {
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(20 * time.Millisecond)
	})
	_, lines := serve(t, h, "GET", "/")

	// slog's JSON handler writes a time.Duration as integer nanoseconds.
	d, ok := only(t, lines)["duration"].(float64)
	if !ok {
		t.Fatalf("duration = %v, want a number of nanoseconds", lines[0]["duration"])
	}
	if got := time.Duration(d); got < 20*time.Millisecond || got > 2*time.Second {
		t.Errorf("duration = %v, want about 20ms", got)
	}
}

func TestLoggerDoesNotChangeTheResponse(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Shift", "morning")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})
	rec, _ := serve(t, h, "PUT", "/")

	if rec.Code != http.StatusCreated {
		t.Errorf("client got status %d, want 201", rec.Code)
	}
	if got := rec.Header().Get("X-Shift"); got != "morning" {
		t.Errorf("client got header X-Shift = %q, want morning", got)
	}
	if got := rec.Body.String(); got != "created" {
		t.Errorf("client got body %q, want created", got)
	}
}

func TestLoggerSeesThe500FromRecover(t *testing.T) {
	// The server wraps Logger around Recover, so a panic is logged by
	// Recover (with the stack) and then by Logger as a 500.
	panicky := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	h := Recover(slog.New(slog.DiscardHandler))(panicky)
	_, lines := serve(t, h, "GET", "/")

	if got := only(t, lines)["status"]; got != float64(500) {
		t.Errorf("status = %v, want 500", got)
	}
}
