package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func discard() *slog.Logger { return slog.New(slog.DiscardHandler) }

// fakeDB is a Pinger whose Ping returns err.
type fakeDB struct{ err error }

func (f fakeDB) Ping(ctx context.Context) error { return f.err }

func TestHealthzWithTheDatabaseUp(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(discard(), fakeDB{}).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["status"] != "ok" {
		t.Errorf("body = %q, want {\"status\":\"ok\"}", rec.Body.String())
	}
}

func TestHealthzWithTheDatabaseDown(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	db := fakeDB{err: errors.New("dial tcp 10.0.0.5:5432: connection refused")}
	rec := httptest.NewRecorder()
	Handler(log, db).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/healthz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["status"] != "unavailable" {
		t.Errorf("body = %q, want {\"status\":\"unavailable\"}", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "10.0.0.5") {
		t.Errorf("the response shows the error to anyone who asks: %s", rec.Body.String())
	}
	if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "connection refused") {
		t.Errorf("want the error logged as a warning; logged:\n%s", logs.String())
	}
}

func TestUnknownAPIRouteIsAProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(discard(), fakeDB{}).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Result().Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}

// listen opens a listener on a free port on localhost.
func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func TestRunLetsARequestInFlightFinish(t *testing.T) {
	started := make(chan struct{})
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(200 * time.Millisecond)
		io.WriteString(w, "done")
	})
	ln := listen(t)
	ctx, stop := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- Run(ctx, New(slow, discard()), ln, 5*time.Second) }()

	// Start a request, and stop the server while it's being handled.
	type result struct {
		body string
		err  error
	}
	resc := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String())
		if err != nil {
			resc <- result{err: err}
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		resc <- result{string(b), err}
	}()
	<-started
	stop()

	if res := <-resc; res.err != nil || res.body != "done" {
		t.Errorf("in-flight request got %q, %v; want done, nil", res.body, res.err)
	}
	if err := <-runErr; err != nil {
		t.Errorf("Run returned %v, want nil after a clean shutdown", err)
	}
	if _, err := http.Get("http://" + ln.Addr().String()); err == nil {
		t.Error("server still accepts connections after shutdown")
	}
}

func TestRunGivesUpAfterTheShutdownTimeout(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	stuck := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	})
	ln := listen(t)
	ctx, stop := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- Run(ctx, New(stuck, discard()), ln, 50*time.Millisecond) }()

	go http.Get("http://" + ln.Addr().String())
	<-started
	stop()

	select {
	case err := <-runErr:
		if err == nil {
			t.Error("Run returned nil, want a deadline error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run didn't give up after its shutdown timeout")
	}
}
