// Package stacks manages compose stack folders on disk.
package stacks

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var (
	ErrNotFound    = errors.New("stack not found")
	ErrExists      = errors.New("stack already exists")
	ErrInvalidName = errors.New("invalid stack name")
)

// composeFiles lists accepted compose file names in lookup order.
var composeFiles = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ValidName reports whether name is a safe stack / compose project name.
func ValidName(name string) bool {
	return len(name) <= 64 && nameRe.MatchString(name)
}

type Store struct {
	Root string
}

type Stack struct {
	Name        string
	Compose     string
	Env         string
	ComposeFile string
}

func (s *Store) Dir(name string) string {
	return filepath.Join(s.Root, name)
}

// composeFile returns the existing compose file name in dir, or "" if none.
func composeFile(dir string) string {
	for _, f := range composeFiles {
		if fi, err := os.Stat(filepath.Join(dir, f)); err == nil && fi.Mode().IsRegular() {
			return f
		}
	}
	return ""
}

// List returns the sorted names of folders that contain a compose file.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() && ValidName(e.Name()) && composeFile(s.Dir(e.Name())) != "" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func (s *Store) Get(name string) (Stack, error) {
	if !ValidName(name) {
		return Stack{}, ErrInvalidName
	}
	dir := s.Dir(name)
	file := composeFile(dir)
	if file == "" {
		return Stack{}, ErrNotFound
	}
	compose, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return Stack{}, err
	}
	env, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Stack{}, err
	}
	return Stack{Name: name, Compose: string(compose), Env: string(env), ComposeFile: file}, nil
}

func (s *Store) Create(name, compose, env string) error {
	if !ValidName(name) {
		return ErrInvalidName
	}
	dir := s.Dir(name)
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ErrExists
		}
		return err
	}
	return writeFiles(dir, composeFiles[0], compose, env)
}

func (s *Store) Save(name, compose, env string) error {
	if !ValidName(name) {
		return ErrInvalidName
	}
	dir := s.Dir(name)
	file := composeFile(dir)
	if file == "" {
		return ErrNotFound
	}
	return writeFiles(dir, file, compose, env)
}

func writeFiles(dir, file, compose, env string) error {
	if err := os.WriteFile(filepath.Join(dir, file), []byte(compose), 0o644); err != nil {
		return err
	}
	envPath := filepath.Join(dir, ".env")
	if env == "" {
		if err := os.Remove(envPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	return os.WriteFile(envPath, []byte(env), 0o644)
}

func (s *Store) Delete(name string) error {
	if !ValidName(name) {
		return ErrInvalidName
	}
	dir := s.Dir(name)
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return err
	}
	return os.RemoveAll(dir)
}
