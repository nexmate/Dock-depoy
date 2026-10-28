package auth

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSetupAndVerify(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "auth.json")}
	if s.IsSetup() {
		t.Fatal("fresh store should not be set up")
	}
	if err := s.Setup("admin", "short"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("short password: %v", err)
	}
	if err := s.Setup("admin", "correct-horse"); err != nil {
		t.Fatal(err)
	}
	if !s.IsSetup() {
		t.Fatal("should be set up")
	}
	if err := s.Setup("other", "another-pass"); !errors.Is(err, ErrAlreadySetup) {
		t.Fatalf("second setup: %v", err)
	}
	if err := s.Verify("admin", "correct-horse"); err != nil {
		t.Fatalf("verify good: %v", err)
	}
	if err := s.Verify("admin", "wrong-pass"); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("verify wrong password: %v", err)
	}
	if err := s.Verify("nobody", "correct-horse"); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("verify wrong user: %v", err)
	}
	// A fresh Store reading the same file sees the credentials.
	s2 := &Store{Path: s.Path}
	if err := s2.Verify("admin", "correct-horse"); err != nil {
		t.Fatalf("reload verify: %v", err)
	}
}

func TestSessionsExpire(t *testing.T) {
	now := time.Unix(1000, 0)
	s := NewSessions(time.Hour)
	s.now = func() time.Time { return now }
	tok := s.Create()
	if len(tok) < 32 || !s.Valid(tok) {
		t.Fatalf("new token should be valid: %q", tok)
	}
	if s.Valid("bogus") || s.Valid("") {
		t.Fatal("unknown token valid")
	}
	now = now.Add(61 * time.Minute)
	if s.Valid(tok) {
		t.Fatal("token should expire")
	}
	tok2 := s.Create()
	s.Delete(tok2)
	if s.Valid(tok2) {
		t.Fatal("deleted token valid")
	}
}

func TestLimiter(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NewLimiter(5, time.Minute, func() time.Time { return now })
	for i := 0; i < 5; i++ {
		if !l.Allow() {
			t.Fatalf("blocked after %d failures", i)
		}
		l.Fail()
	}
	if l.Allow() {
		t.Fatal("should block after 5 failures")
	}
	now = now.Add(61 * time.Second)
	if !l.Allow() {
		t.Fatal("should allow after window")
	}
}
