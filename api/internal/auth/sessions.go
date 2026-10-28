package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Sessions is an in-memory session token store; restarting the API logs everyone out.
type Sessions struct {
	TTL time.Duration

	mu      sync.Mutex
	expires map[string]time.Time
	now     func() time.Time
}

func NewSessions(ttl time.Duration) *Sessions {
	return &Sessions{TTL: ttl, expires: map[string]time.Time{}, now: time.Now}
}

func (s *Sessions) Create() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	tok := hex.EncodeToString(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for t, exp := range s.expires {
		if now.After(exp) {
			delete(s.expires, t)
		}
	}
	s.expires[tok] = now.Add(s.TTL)
	return tok
}

func (s *Sessions) Valid(token string) bool {
	if token == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.expires[token]
	if !ok {
		return false
	}
	if s.now().After(exp) {
		delete(s.expires, token)
		return false
	}
	return true
}

func (s *Sessions) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.expires, token)
}
