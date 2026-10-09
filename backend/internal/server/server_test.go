package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func discard() *slog.Logger { return slog.New(slog.DiscardHandler) }

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(discard()).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["status"] != "ok" {
		t.Errorf("body = %q, want {\"status\":\"ok\"}", rec.Body.String())
	}
}

func TestUnknownAPIRouteIsAProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(discard()).ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
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
