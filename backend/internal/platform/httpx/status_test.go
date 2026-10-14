package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusRecorderNotesTheStatus(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	rec.WriteHeader(http.StatusNotFound)
	if rec.Status() != http.StatusNotFound {
		t.Errorf("Status() = %d, want 404", rec.Status())
	}
}

func TestStatusRecorderKeepsTheFirstStatus(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	rec.WriteHeader(http.StatusAccepted)
	rec.WriteHeader(http.StatusInternalServerError)
	if rec.Status() != http.StatusAccepted {
		t.Errorf("Status() = %d, want 202", rec.Status())
	}
}

func TestStatusRecorderWriteMeans200(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	rec.Write([]byte("hello"))
	if rec.Status() != http.StatusOK {
		t.Errorf("Status() = %d, want 200", rec.Status())
	}
}

func TestStatusRecorderNothingSentMeans200(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	if rec.Status() != http.StatusOK {
		t.Errorf("Status() = %d, want 200", rec.Status())
	}
}

func TestStatusRecorderPassesTheResponseThrough(t *testing.T) {
	real := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: real}
	rec.Header().Set("X-Shift", "morning")
	rec.WriteHeader(http.StatusCreated)
	rec.Write([]byte("created"))

	got := real.Result()
	if got.StatusCode != http.StatusCreated || got.Header.Get("X-Shift") != "morning" || real.Body.String() != "created" {
		t.Errorf("client got %d, X-Shift=%q, body %q; want 201, morning, created",
			got.StatusCode, got.Header.Get("X-Shift"), real.Body.String())
	}
}
