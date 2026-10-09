package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusCreated, map[string]string{"status": "ok"})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	// rec.Result().Header is what the client received: the headers as they
	// were when the status was written. rec.Header() is the live map, which
	// would also show a header set too late to be sent.
	if ct := rec.Result().Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if got := rec.Body.String(); got != `{"status":"ok"}`+"\n" {
		t.Errorf("body = %q", got)
	}
}

func TestJSONThatCantBeEncodedIsA500(t *testing.T) {
	// JSON logs the encoding error through slog's default logger; keep it
	// out of the test output.
	defer slog.SetDefault(slog.Default())
	slog.SetDefault(slog.New(slog.DiscardHandler))

	rec := httptest.NewRecorder()
	JSON(rec, http.StatusOK, map[string]any{"oops": make(chan int)})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if ct := rec.Result().Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestError(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, http.StatusNotFound, "no route for GET /api/v1/nope")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Result().Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q", ct)
	}
	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("body isn't JSON: %v", err)
	}
	want := Problem{Title: "Not Found", Status: 404, Detail: "no route for GET /api/v1/nope"}
	if p != want {
		t.Errorf("got %+v, want %+v", p, want)
	}
}
