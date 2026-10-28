package compose

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrBusy          = errors.New("another action is already running for this stack")
	ErrUnknownAction = errors.New("unknown action")
)

type Event struct {
	Line     string
	Done     bool
	ExitCode int
}

// Job is one stack action. Output is buffered so late subscribers replay from the start.
type Job struct {
	ID, Stack, Action string

	mu     sync.Mutex
	lines  []string
	done   bool
	exit   int
	notify chan struct{}
}

func newJob(stack, action string) *Job {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return &Job{ID: hex.EncodeToString(b), Stack: stack, Action: action, notify: make(chan struct{})}
}

func (j *Job) wake() {
	close(j.notify)
	j.notify = make(chan struct{})
}

func (j *Job) append(line string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.lines = append(j.lines, line)
	j.wake()
}

func (j *Job) finish(code int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.done, j.exit = true, code
	j.wake()
}

// Subscribe streams all lines so far, then live lines, then a final Done event.
// The channel closes after Done or when ctx is cancelled.
func (j *Job) Subscribe(ctx context.Context) <-chan Event {
	ch := make(chan Event, 64)
	go func() {
		defer close(ch)
		i := 0
		for {
			j.mu.Lock()
			pending := j.lines[i:]
			done, exit, wait := j.done, j.exit, j.notify
			j.mu.Unlock()
			for _, l := range pending {
				select {
				case ch <- Event{Line: l}:
				case <-ctx.Done():
					return
				}
			}
			i += len(pending)
			if done {
				select {
				case ch <- Event{Done: true, ExitCode: exit}:
				case <-ctx.Done():
				}
				return
			}
			select {
			case <-wait:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

// Manager starts jobs, allowing one running action per stack.
type Manager struct {
	runner    Runner
	retention time.Duration

	mu   sync.Mutex
	busy map[string]bool
	jobs map[string]*Job
}

func NewManager(r Runner, retention time.Duration) *Manager {
	return &Manager{runner: r, retention: retention, busy: map[string]bool{}, jobs: map[string]*Job{}}
}

func (m *Manager) lock(stack string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.busy[stack] {
		return false
	}
	m.busy[stack] = true
	return true
}

func (m *Manager) unlock(stack string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.busy, stack)
}

// Start launches action for stack in dir and returns immediately.
func (m *Manager) Start(stack, dir, action string) (*Job, error) {
	cmds, ok := ActionArgs(action)
	if !ok {
		return nil, ErrUnknownAction
	}
	if !m.lock(stack) {
		return nil, ErrBusy
	}
	j := newJob(stack, action)
	m.mu.Lock()
	m.jobs[j.ID] = j
	m.mu.Unlock()
	go func() {
		code := 0
		for _, args := range cmds {
			j.append("$ docker compose " + strings.Join(args, " "))
			c, err := m.runner.Run(context.Background(), dir, j.append, args...)
			if err != nil {
				j.append("error: " + err.Error())
				if c == 0 {
					c = -1
				}
			}
			if code = c; code != 0 {
				break
			}
		}
		m.unlock(stack)
		j.finish(code)
		time.AfterFunc(m.retention, func() {
			m.mu.Lock()
			delete(m.jobs, j.ID)
			m.mu.Unlock()
		})
	}()
	return j, nil
}

func (m *Manager) Get(id string) (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	return j, ok
}

// RunSync runs one compose command under the stack lock and waits for it.
func (m *Manager) RunSync(ctx context.Context, stack, dir string, args ...string) error {
	if !m.lock(stack) {
		return ErrBusy
	}
	defer m.unlock(stack)
	var buf strings.Builder
	code, err := m.runner.Run(ctx, dir, func(l string) { buf.WriteString(l + "\n") }, args...)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("exit %d: %s", code, strings.TrimSpace(buf.String()))
	}
	return nil
}
