package compose

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRunner records calls, emits scripted lines and can block until released.
type fakeRunner struct {
	mu      sync.Mutex
	calls   [][]string
	codes   []int         // exit code per call, default 0
	release chan struct{} // if non-nil, Run blocks until closed
	lines   []string
}

func (f *fakeRunner) Run(ctx context.Context, dir string, out func(string), args ...string) (int, error) {
	f.mu.Lock()
	n := len(f.calls)
	f.calls = append(f.calls, args)
	f.mu.Unlock()
	for _, l := range f.lines {
		out(l)
	}
	if f.release != nil {
		<-f.release
	}
	if n < len(f.codes) {
		return f.codes[n], nil
	}
	return 0, nil
}

func collect(t *testing.T, j *Job) ([]string, Event) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var lines []string
	for ev := range j.Subscribe(ctx) {
		if ev.Done {
			return lines, ev
		}
		lines = append(lines, ev.Line)
	}
	t.Fatal("stream closed without Done")
	return nil, Event{}
}

func TestLateSubscriberReplaysAll(t *testing.T) {
	r := &fakeRunner{lines: []string{"a", "b"}, codes: []int{3}}
	m := NewManager(r, time.Minute)
	j, err := m.Start("web", "/x", "up")
	if err != nil {
		t.Fatal(err)
	}
	// wait for completion, then subscribe
	collect(t, j)
	lines, done := collect(t, j)
	if !reflect.DeepEqual(lines, []string{"$ docker compose up -d --remove-orphans", "a", "b"}) || done.ExitCode != 3 {
		t.Fatalf("replay got %v %+v", lines, done)
	}
	if got, ok := m.Get(j.ID); !ok || got != j {
		t.Fatal("Get should find job")
	}
}

func TestBusyStack(t *testing.T) {
	r := &fakeRunner{release: make(chan struct{})}
	m := NewManager(r, time.Minute)
	j, err := m.Start("web", "/x", "up")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start("web", "/x", "stop"); !errors.Is(err, ErrBusy) {
		t.Fatalf("second start = %v, want ErrBusy", err)
	}
	if err := m.RunSync(context.Background(), "web", "/x", "down"); !errors.Is(err, ErrBusy) {
		t.Fatalf("RunSync on busy = %v", err)
	}
	other, err := m.Start("db", "/y", "up")
	if err != nil {
		t.Fatalf("other stack should run: %v", err)
	}
	close(r.release)
	collect(t, j)
	collect(t, other)
	if _, err := m.Start("web", "/x", "stop"); err != nil {
		t.Fatalf("lock not released: %v", err)
	}
}

func TestPullStopsOnFailure(t *testing.T) {
	r := &fakeRunner{codes: []int{1, 0}}
	m := NewManager(r, time.Minute)
	j, _ := m.Start("web", "/x", "pull")
	_, done := collect(t, j)
	if done.ExitCode != 1 || len(r.calls) != 1 {
		t.Fatalf("exit=%d calls=%v", done.ExitCode, r.calls)
	}

	r2 := &fakeRunner{}
	m2 := NewManager(r2, time.Minute)
	j2, _ := m2.Start("web", "/x", "pull")
	collect(t, j2)
	want := [][]string{{"pull"}, {"up", "-d", "--remove-orphans"}}
	if !reflect.DeepEqual(r2.calls, want) {
		t.Fatalf("calls = %v", r2.calls)
	}
}

func TestActionArgs(t *testing.T) {
	if _, ok := ActionArgs("bogus"); ok {
		t.Fatal("bogus action accepted")
	}
	for _, a := range []string{"up", "stop", "restart", "pull", "down"} {
		if _, ok := ActionArgs(a); !ok {
			t.Fatalf("%s rejected", a)
		}
	}
	if _, err := NewManager(&fakeRunner{}, time.Minute).Start("web", "/x", "bogus"); !errors.Is(err, ErrUnknownAction) {
		t.Fatalf("Start bogus = %v", err)
	}
}

func TestJobEviction(t *testing.T) {
	m := NewManager(&fakeRunner{}, 10*time.Millisecond)
	j, _ := m.Start("web", "/x", "up")
	collect(t, j)
	time.Sleep(50 * time.Millisecond)
	if _, ok := m.Get(j.ID); ok {
		t.Fatal("job should be evicted")
	}
}

func TestRunSyncReportsExitCode(t *testing.T) {
	m := NewManager(&fakeRunner{codes: []int{2}, lines: []string{"boom"}}, time.Minute)
	err := m.RunSync(context.Background(), "web", "/x", "down")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("RunSync = %v", err)
	}
}

// validateRunner fails when the compose file it is pointed at contains "bad".
type validateRunner struct{ args []string }

func (v *validateRunner) Run(ctx context.Context, dir string, out func(string), args ...string) (int, error) {
	v.args = args
	b, err := os.ReadFile(filepath.Join(dir, args[1]))
	if err != nil {
		return 1, err
	}
	if strings.Contains(string(b), "bad") {
		out("yaml: line 1: did not find expected key")
		return 15, nil
	}
	return 0, nil
}

func TestValidate(t *testing.T) {
	dir := t.TempDir()
	v := &validateRunner{}
	if err := Validate(context.Background(), v, dir, "services: {}", "A=1"); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if !reflect.DeepEqual(v.args, []string{"-f", ".dd-validate.yaml", "--env-file", ".dd-validate.env", "config", "-q"}) {
		t.Fatalf("args = %v", v.args)
	}
	err := Validate(context.Background(), v, dir, "bad", "")
	if err == nil || !strings.Contains(err.Error(), "did not find expected key") {
		t.Fatalf("invalid: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("temp files left behind: %v", entries)
	}
	// Missing dir (new stack) still validates, in a throwaway dir.
	if err := Validate(context.Background(), v, filepath.Join(dir, "new"), "services: {}", ""); err != nil {
		t.Fatalf("missing dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new")); !os.IsNotExist(err) {
		t.Fatal("validate must not create the stack dir")
	}
}
