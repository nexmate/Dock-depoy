package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// heartbeatInterval is how often idle SSE streams get a comment line to keep proxies from timing out.
var heartbeatInterval = 15 * time.Second

type sseWriter struct {
	mu sync.Mutex
	w  http.ResponseWriter
	f  http.Flusher
}

// newSSE starts an event stream and a heartbeat; call the returned stop before the handler returns.
func newSSE(w http.ResponseWriter) (*sseWriter, func()) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	f, _ := w.(http.Flusher)
	s := &sseWriter{w: w, f: f}
	s.write(": connected\n\n")

	quit, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(heartbeatInterval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				s.write(": ping\n\n")
			case <-quit:
				return
			}
		}
	}()
	return s, func() {
		close(quit)
		<-done
	}
}

func (s *sseWriter) write(raw string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprint(s.w, raw)
	if s.f != nil {
		s.f.Flush()
	}
}

// send writes one event whose data is v encoded as JSON.
func (s *sseWriter) send(event string, v any) {
	b, _ := json.Marshal(v)
	s.write(fmt.Sprintf("event: %s\ndata: %s\n\n", event, b))
}
