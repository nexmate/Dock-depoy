package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nenad/docker-depoy/api/internal/auth"
	"github.com/nenad/docker-depoy/api/internal/compose"
	"github.com/nenad/docker-depoy/api/internal/docker"
	"github.com/nenad/docker-depoy/api/internal/stacks"
)

// fakeRunner: `config` fails when the file under test contains "bad";
// `down` exits with downCode; anything else prints "ok" and blocks on hold if set.
type fakeRunner struct {
	downCode int
	hold     chan struct{}
}

func (f *fakeRunner) Run(ctx context.Context, dir string, out func(string), args ...string) (int, error) {
	if len(args) >= 2 && args[0] == "-f" {
		b, _ := os.ReadFile(filepath.Join(dir, args[1]))
		if strings.Contains(string(b), "bad") {
			out("yaml: invalid compose")
			return 15, nil
		}
		return 0, nil
	}
	if args[0] == "down" {
		return f.downCode, nil
	}
	out("ok")
	if f.hold != nil {
		<-f.hold
	}
	return 0, nil
}

type fakeDocker struct{ containers map[string][]docker.Container }

func (f *fakeDocker) ProjectContainers(ctx context.Context) (map[string][]docker.Container, error) {
	return f.containers, nil
}

func (f *fakeDocker) Logs(ctx context.Context, id string, tail int, out func(string)) error {
	out("log line 1")
	out("log line 2")
	return nil
}

type harness struct {
	t      *testing.T
	h      http.Handler
	deps   Deps
	runner *fakeRunner
	cookie *http.Cookie
}

func newHarness(t *testing.T) *harness {
	dir := t.TempDir()
	r := &fakeRunner{}
	deps := Deps{
		Stacks:   &stacks.Store{Root: filepath.Join(dir, "stacks")},
		Auth:     &auth.Store{Path: filepath.Join(dir, "data", "auth.json")},
		Sessions: auth.NewSessions(time.Hour),
		Limiter:  auth.NewLimiter(5, time.Minute, nil),
		Jobs:     compose.NewManager(r, time.Minute),
		Runner:   r,
		Docker: &fakeDocker{containers: map[string][]docker.Container{
			"web": {{ID: "c1", Name: "web-app-1", Service: "app", State: "running", Ports: []docker.Port{{Public: 8080, Private: 80, Proto: "tcp"}}}},
		}},
	}
	return &harness{t: t, h: New(deps), deps: deps, runner: r}
}

func (h *harness) do(method, path, body string) *httptest.ResponseRecorder {
	h.t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if h.cookie != nil {
		req.AddCookie(h.cookie)
	}
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	return rec
}

func (h *harness) login() {
	h.t.Helper()
	rec := h.do("POST", "/api/auth/setup", `{"username":"admin","password":"password123"}`)
	if rec.Code != 200 {
		h.t.Fatalf("setup: %d %s", rec.Code, rec.Body)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			h.cookie = c
		}
	}
	if h.cookie == nil || !h.cookie.HttpOnly || h.cookie.SameSite != http.SameSiteStrictMode {
		h.t.Fatalf("bad session cookie: %+v", h.cookie)
	}
}

func expect(t *testing.T, rec *httptest.ResponseRecorder, code int) {
	t.Helper()
	if rec.Code != code {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, code, rec.Body)
	}
}

func TestRequiresAuth(t *testing.T) {
	h := newHarness(t)
	expect(t, h.do("GET", "/api/stacks", ""), 401)
	rec := h.do("GET", "/api/auth/status", "")
	expect(t, rec, 200)
	if !strings.Contains(rec.Body.String(), `"setupDone":false`) {
		t.Fatalf("status body %s", rec.Body)
	}
}

func TestSetupLoginLogout(t *testing.T) {
	h := newHarness(t)
	h.login()
	expect(t, h.do("POST", "/api/auth/setup", `{"username":"x","password":"password123"}`), 409)
	rec := h.do("GET", "/api/auth/status", "")
	if !strings.Contains(rec.Body.String(), `"authenticated":true`) {
		t.Fatalf("status %s", rec.Body)
	}
	expect(t, h.do("POST", "/api/auth/logout", ""), 204)
	expect(t, h.do("GET", "/api/stacks", ""), 401)
	h.cookie = nil
	expect(t, h.do("POST", "/api/auth/login", `{"username":"admin","password":"password123"}`), 200)
}

func TestLoginRateLimit(t *testing.T) {
	h := newHarness(t)
	h.login()
	h.cookie = nil
	for i := 0; i < 5; i++ {
		expect(t, h.do("POST", "/api/auth/login", `{"username":"admin","password":"wrong-pass"}`), 401)
	}
	expect(t, h.do("POST", "/api/auth/login", `{"username":"admin","password":"password123"}`), 429)
}

func TestWeakPasswordSetup(t *testing.T) {
	h := newHarness(t)
	expect(t, h.do("POST", "/api/auth/setup", `{"username":"admin","password":"short"}`), 400)
}

func TestStackCRUD(t *testing.T) {
	h := newHarness(t)
	h.login()
	expect(t, h.do("POST", "/api/stacks", `{"name":"../x","compose":"services: {}"}`), 400)

	rec := h.do("POST", "/api/stacks", `{"name":"web","compose":"bad"}`)
	expect(t, rec, 422)
	if !strings.Contains(rec.Body.String(), "yaml: invalid compose") {
		t.Fatalf("422 should carry compose output: %s", rec.Body)
	}
	if _, err := os.Stat(h.deps.Stacks.Dir("web")); !os.IsNotExist(err) {
		t.Fatal("invalid create must not create folder")
	}

	expect(t, h.do("POST", "/api/stacks", `{"name":"web","compose":"services: {}","env":"A=1"}`), 201)
	expect(t, h.do("POST", "/api/stacks", `{"name":"web","compose":"services: {}"}`), 409)

	rec = h.do("GET", "/api/stacks/web", "")
	expect(t, rec, 200)
	var st struct{ Name, Compose, Env, Status string }
	_ = json.Unmarshal(rec.Body.Bytes(), &st)
	if st.Compose != "services: {}" || st.Env != "A=1" || st.Status != "running" {
		t.Fatalf("get = %+v", st)
	}

	expect(t, h.do("PUT", "/api/stacks/web", `{"compose":"bad","env":""}`), 422)
	if s, _ := h.deps.Stacks.Get("web"); s.Compose != "services: {}" {
		t.Fatalf("invalid save changed file: %q", s.Compose)
	}
	expect(t, h.do("PUT", "/api/stacks/web", `{"compose":"services: {a: {}}","env":""}`), 200)

	rec = h.do("GET", "/api/stacks", "")
	expect(t, rec, 200)
	if !strings.Contains(rec.Body.String(), `"name":"web","status":"running","containers":{"running":1,"total":1}`) {
		t.Fatalf("list = %s", rec.Body)
	}
	expect(t, h.do("GET", "/api/stacks/nope", ""), 404)
}

func TestDeleteNeverDeployed(t *testing.T) {
	h := newHarness(t)
	h.login()
	h.runner.downCode = 1
	expect(t, h.do("POST", "/api/stacks", `{"name":"fresh","compose":"services: {}"}`), 201)
	expect(t, h.do("DELETE", "/api/stacks/fresh", ""), 204)
	if _, err := os.Stat(h.deps.Stacks.Dir("fresh")); !os.IsNotExist(err) {
		t.Fatal("folder should be removed")
	}
	expect(t, h.do("DELETE", "/api/stacks/fresh", ""), 404)
}

func TestActionAndStream(t *testing.T) {
	h := newHarness(t)
	h.login()
	expect(t, h.do("POST", "/api/stacks", `{"name":"web","compose":"services: {}"}`), 201)
	expect(t, h.do("POST", "/api/stacks/web/bogus", ""), 400)
	expect(t, h.do("POST", "/api/stacks/nope/up", ""), 404)

	h.runner.hold = make(chan struct{})
	rec := h.do("POST", "/api/stacks/web/up", "")
	expect(t, rec, 202)
	var job struct{ JobID string }
	_ = json.Unmarshal(rec.Body.Bytes(), &job)
	expect(t, h.do("POST", "/api/stacks/web/stop", ""), 409)
	close(h.runner.hold)

	rec = h.do("GET", "/api/jobs/"+job.JobID+"/stream", "")
	expect(t, rec, 200)
	body := rec.Body.String()
	if rec.Header().Get("Content-Type") != "text/event-stream" ||
		!strings.Contains(body, "event: line\ndata: \"ok\"\n\n") ||
		!strings.Contains(body, "event: done\ndata: {\"exitCode\":0}\n\n") {
		t.Fatalf("stream = %q", body)
	}
	expect(t, h.do("GET", "/api/jobs/missing/stream", ""), 404)
}

func TestContainersAndLogs(t *testing.T) {
	h := newHarness(t)
	h.login()
	expect(t, h.do("POST", "/api/stacks", `{"name":"web","compose":"services: {}"}`), 201)
	rec := h.do("GET", "/api/stacks/web/containers", "")
	expect(t, rec, 200)
	if !strings.Contains(rec.Body.String(), `"ports":[{"public":8080,"private":80,"proto":"tcp"}]`) {
		t.Fatalf("containers = %s", rec.Body)
	}
	rec = h.do("GET", "/api/containers/c1/logs?tail=10", "")
	expect(t, rec, 200)
	if !strings.Contains(rec.Body.String(), "event: line\ndata: \"log line 2\"\n\n") {
		t.Fatalf("logs = %q", rec.Body)
	}
}

func TestRejectsCrossOriginWrites(t *testing.T) {
	h := newHarness(t)
	h.login()
	expect(t, h.do("POST", "/api/stacks", `{"name":"web","compose":"services: {}"}`), 201)
	req := httptest.NewRequest("POST", "/api/stacks/web/down", nil)
	req.AddCookie(h.cookie)
	req.Header.Set("Sec-Fetch-Site", "same-site")
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	expect(t, rec, 403)
}

func TestRequiresJSONContentType(t *testing.T) {
	h := newHarness(t)
	h.login()
	req := httptest.NewRequest("POST", "/api/stacks", strings.NewReader(`{"name":"web","compose":"services: {}"}`))
	req.AddCookie(h.cookie)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	expect(t, rec, 415)
}

func TestJobStreamHeartbeat(t *testing.T) {
	old := heartbeatInterval
	heartbeatInterval = 5 * time.Millisecond
	defer func() { heartbeatInterval = old }()
	h := newHarness(t)
	h.login()
	expect(t, h.do("POST", "/api/stacks", `{"name":"web","compose":"services: {}"}`), 201)
	h.runner.hold = make(chan struct{})
	rec := h.do("POST", "/api/stacks/web/up", "")
	var job struct{ JobID string }
	_ = json.Unmarshal(rec.Body.Bytes(), &job)
	go func() { time.Sleep(60 * time.Millisecond); close(h.runner.hold) }()
	rec = h.do("GET", "/api/jobs/"+job.JobID+"/stream", "")
	if !strings.Contains(rec.Body.String(), ": ping\n\n") {
		t.Fatalf("no heartbeat in %q", rec.Body)
	}
}

func TestDeleteKeepsFolderWhenDownFailsWithContainers(t *testing.T) {
	h := newHarness(t)
	h.login()
	h.runner.downCode = 1 // fake docker reports a running container for "web"
	expect(t, h.do("POST", "/api/stacks", `{"name":"web","compose":"services: {}"}`), 201)
	expect(t, h.do("DELETE", "/api/stacks/web", ""), 500)
	if _, err := os.Stat(h.deps.Stacks.Dir("web")); err != nil {
		t.Fatal("folder must be kept when containers remain")
	}
}
