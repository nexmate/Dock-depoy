package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type sseWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

func newSSE(w http.ResponseWriter) *sseWriter {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	f, _ := w.(http.Flusher)
	s := &sseWriter{w: w, f: f}
	s.flush()
	return s
}

func (s *sseWriter) flush() {
	if s.f != nil {
		s.f.Flush()
	}
}

// send writes one event whose data is v encoded as JSON.
func (s *sseWriter) send(event string, v any) {
	b, _ := json.Marshal(v)
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, b)
	s.flush()
}
