// Package auth holds the single admin credential, sessions and login rate limiting.
package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAlreadySetup   = errors.New("admin account already exists")
	ErrBadCredentials = errors.New("invalid username or password")
	ErrWeakPassword   = errors.New("password must be at least 8 characters")
)

const CookieName = "dd_session"

type credentials struct {
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
}

// Store persists the admin credential as JSON at Path.
type Store struct {
	Path string
	mu   sync.Mutex
}

func (s *Store) load() (*credentials, error) {
	b, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) IsSetup() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.load()
	return err == nil && c != nil
}

func (s *Store) Setup(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, err := s.load(); err != nil {
		return err
	} else if c != nil {
		return ErrAlreadySetup
	}
	if username == "" {
		return ErrBadCredentials
	}
	if len(password) < 8 {
		return ErrWeakPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(credentials{Username: username, PasswordHash: string(hash)})
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.Path, b, 0o600)
}

func (s *Store) Verify(username, password string) error {
	s.mu.Lock()
	c, err := s.load()
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if c == nil {
		return ErrBadCredentials
	}
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(c.Username)) == 1
	passErr := bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(password))
	if !userOK || passErr != nil {
		return ErrBadCredentials
	}
	return nil
}
