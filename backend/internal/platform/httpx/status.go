package httpx

import "net/http"

// statusRecorder sits between a handler and the real response, and writes
// down the status code the handler sends.
//
// Why it's needed: a handler sends its response by calling methods on an
// http.ResponseWriter (WriteHeader to send a status code, Write to send body
// bytes). Once they're sent, there's no way to ask the ResponseWriter which
// status it was. So Logger hands the handler this recorder instead: it looks
// like a ResponseWriter, passes everything through to the real one, and keeps
// a note of the status on the way.
//
// How it looks like a ResponseWriter: the http.ResponseWriter field has no
// name, which Go calls embedding. statusRecorder gets all of that field's
// methods for free, and only the two below are replaced with versions that
// take notes first.
type statusRecorder struct {
	http.ResponseWriter
	status int // 0 until the handler sends something
}

// WriteHeader notes the status, then sends it. Only the first status counts:
// that's the one the client gets.
func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

// Write sends body bytes. Writing a body without choosing a status first
// sends 200 OK, so that's what gets noted.
func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// Status is the status code the handler sent. A handler that sent nothing at
// all still produces a 200 OK response, so that's the answer then too.
func (s *statusRecorder) Status() int {
	if s.status == 0 {
		return http.StatusOK
	}
	return s.status
}
