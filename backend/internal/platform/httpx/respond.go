// Package httpx holds the HTTP pieces every feature shares: writing JSON and
// errors, and the middleware that wraps every request.
package httpx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

// JSON writes v as a JSON response with the given status.
//
// It encodes into a buffer first: if encoding fails, nothing has been sent
// yet, so the client gets a clean 500 instead of a 200 with half a body.
func JSON(w http.ResponseWriter, status int, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		slog.Error("encoding JSON response", "err", err)
		Error(w, http.StatusInternalServerError, "")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

// Problem is an error response in the RFC 9457 format
// ("Problem Details for HTTP APIs"): one shape for every error the API
// returns, which the frontend can rely on.
type Problem struct {
	// Type identifies the kind of problem; empty means "about:blank", that
	// is, nothing more specific than the status code.
	Type   string `json:"type,omitempty"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Error writes a Problem with the status's standard title and an optional
// detail. Details are for people reading responses; code should branch on
// the status (and, later, Type).
func Error(w http.ResponseWriter, status int, detail string) {
	body, _ := json.Marshal(Problem{Title: http.StatusText(status), Status: status, Detail: detail})
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	w.Write(append(body, '\n'))
}
