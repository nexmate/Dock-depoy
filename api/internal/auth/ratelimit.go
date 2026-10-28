package auth

import (
	"sync"
	"time"
)

// Limiter counts failed logins in a sliding window.
type Limiter struct {
	max    int
	window time.Duration
	now    func() time.Time

	mu    sync.Mutex
	fails []time.Time
}

func NewLimiter(max int, window time.Duration, now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{max: max, window: window, now: now}
}

func (l *Limiter) prune() {
	cutoff := l.now().Add(-l.window)
	i := 0
	for i < len(l.fails) && !l.fails[i].After(cutoff) {
		i++
	}
	l.fails = l.fails[i:]
}

func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune()
	return len(l.fails) < l.max
}

func (l *Limiter) Fail() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune()
	l.fails = append(l.fails, l.now())
}
