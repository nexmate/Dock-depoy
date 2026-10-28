package stacks

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidName(t *testing.T) {
	for _, n := range []string{"my-app", "a_1", "0x"} {
		if !ValidName(n) {
			t.Errorf("ValidName(%q) = false, want true", n)
		}
	}
	for _, n := range []string{"", "../x", "-a", "_a", "A", "a b", "a/b", strings.Repeat("a", 65)} {
		if ValidName(n) {
			t.Errorf("ValidName(%q) = true, want false", n)
		}
	}
}

func TestCreateGetSave(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	if err := s.Create("web", "services: {}\n", "A=1\n"); err != nil {
		t.Fatal(err)
	}
	st, err := s.Get("web")
	if err != nil {
		t.Fatal(err)
	}
	if st.Compose != "services: {}\n" || st.Env != "A=1\n" || st.ComposeFile != "compose.yaml" {
		t.Fatalf("unexpected stack %+v", st)
	}
	if err := s.Save("web", "services: {x: {}}\n", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "web", ".env")); !os.IsNotExist(err) {
		t.Fatalf(".env should be removed when env empty, stat err=%v", err)
	}
	st, _ = s.Get("web")
	if st.Compose != "services: {x: {}}\n" || st.Env != "" {
		t.Fatalf("unexpected stack after save %+v", st)
	}
}

func TestListSkipsDirsWithoutCompose(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	_ = s.Create("b", "x", "")
	_ = s.Create("a", "x", "")
	_ = os.Mkdir(filepath.Join(s.Root, "stray"), 0o755)
	_ = os.WriteFile(filepath.Join(s.Root, "file.txt"), []byte("x"), 0o644)
	got, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("List = %v", got)
	}
}

func TestGetPrefersComposeYamlOrder(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	dir := filepath.Join(s.Root, "old")
	_ = os.Mkdir(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("legacy"), 0o644)
	st, err := s.Get("old")
	if err != nil || st.Compose != "legacy" || st.ComposeFile != "docker-compose.yml" {
		t.Fatalf("Get = %+v, %v", st, err)
	}
	if err := s.Save("old", "updated", ""); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if string(b) != "updated" {
		t.Fatalf("save should write back to docker-compose.yml, got %q", b)
	}
	if _, err := os.Stat(filepath.Join(dir, "compose.yaml")); !os.IsNotExist(err) {
		t.Fatal("compose.yaml should not be created")
	}
}

func TestErrors(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	_ = s.Create("x", "c", "")
	if err := s.Create("x", "c", ""); !errors.Is(err, ErrExists) {
		t.Errorf("Create existing = %v", err)
	}
	if _, err := s.Get("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get missing = %v", err)
	}
	if err := s.Save("nope", "c", ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("Save missing = %v", err)
	}
	if err := s.Delete("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete missing = %v", err)
	}
	bad := "../evil"
	if _, err := s.Get(bad); !errors.Is(err, ErrInvalidName) {
		t.Errorf("Get bad = %v", err)
	}
	if err := s.Create(bad, "c", ""); !errors.Is(err, ErrInvalidName) {
		t.Errorf("Create bad = %v", err)
	}
	if err := s.Save(bad, "c", ""); !errors.Is(err, ErrInvalidName) {
		t.Errorf("Save bad = %v", err)
	}
	if err := s.Delete(bad); !errors.Is(err, ErrInvalidName) {
		t.Errorf("Delete bad = %v", err)
	}
	if err := s.Delete("x"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Dir("x")); !os.IsNotExist(err) {
		t.Fatal("dir should be gone")
	}
}
