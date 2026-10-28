# Docker-depoy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Single-user web app to create, edit, deploy and observe Docker Compose stacks on one host.

**Architecture:** Go API (`api/`) owns stack files, runs `docker compose` CLI, queries Docker Engine SDK, streams via SSE. Next.js (`web/`) is UI only and proxies `/api/*` to Go via rewrites. Two containers in root `compose.yaml`; only `web` is published.

**Tech Stack:** Go 1.23+ stdlib `net/http`, `github.com/docker/docker/client`, `golang.org/x/crypto/bcrypt`; Next.js App Router + shadcn/ui (preset `b1YmpVh4a`), TanStack Query, next-themes, `@uiw/react-codemirror` + `@codemirror/lang-yaml`, sonner, Vitest.

**Spec:** `docs/superpowers/specs/2026-10-28-docker-depoy-design.md`

## Global Constraints

- Stack name regex `^[a-z0-9][a-z0-9_-]*$`, max 64 chars.
- Compose filename precedence: `compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml`; new stacks write `compose.yaml`.
- Env defaults: `LISTEN_ADDR=:8080`, `STACKS_DIR=/opt/stacks`, `DATA_DIR=/data`, `COOKIE_SECURE=false`, web `API_URL=http://api:8080`.
- Cookie `dd_session`, HttpOnly, SameSite=Strict, Path=/, 7-day expiry; password ≥ 8 chars; 5 failed logins/minute global → 429.
- Error body `{error, detail?}`.
- Theme tokens copied verbatim from spec's user-provided palette into `web/app/globals.css`.
- Web published on host port `5001`.

## Review Focus

- Stack folder with no compose file (stray dir in `STACKS_DIR`) → skipped by List, not an error. Test in Task 1.
- Late SSE subscriber to a finished job → receives full replay then `done`. Test in Task 3.
- Second action on a busy stack → 409, first job unaffected. Test in Task 3.
- Delete of a never-deployed stack → `down` failure ignored, folder removed. Test in Task 5.
- Invalid YAML on save → 422 with compose's stderr in `detail`, file on disk unchanged. Test in Task 5.

---

### Task 1: Go module + `stacks` package

**Files:** Create `api/go.mod`, `api/internal/stacks/stacks.go`, `api/internal/stacks/stacks_test.go`

**Interfaces — Produces:**
```go
var ErrNotFound, ErrExists, ErrInvalidName error
func ValidName(name string) bool
type Store struct{ Root string }
type Stack struct{ Name, Compose, Env, ComposeFile string }
func (s *Store) List() ([]string, error)                 // sorted dir names containing a compose file
func (s *Store) Get(name string) (Stack, error)
func (s *Store) Create(name, compose, env string) error   // ErrExists if dir exists
func (s *Store) Save(name, compose, env string) error     // ErrNotFound if missing; empty env removes .env
func (s *Store) Delete(name string) error
func (s *Store) Dir(name string) string
```

- [ ] Tests: `TestValidName` (accepts `my-app`, `a_1`; rejects `../x`, `-a`, `A`, 65-char name); `TestCreateGetSave` (round-trip, `.env` deleted when env empty); `TestListSkipsDirsWithoutCompose`; `TestGetPrefersComposeYamlOrder` (dir with only `docker-compose.yml` reads it and Save writes back to it); `TestCreateExisting` → `ErrExists`; all methods return `ErrInvalidName` for bad names.
- [ ] Run `go test ./internal/stacks/` → FAIL, implement, → PASS. Commit.

### Task 2: `auth` package

**Files:** Create `api/internal/auth/{store.go,sessions.go,ratelimit.go,auth_test.go}`

**Interfaces — Produces:**
```go
var ErrAlreadySetup, ErrBadCredentials, ErrWeakPassword error
type Store struct{ Path string }                       // DATA_DIR/auth.json
func (s *Store) IsSetup() bool
func (s *Store) Setup(username, password string) error // bcrypt; ErrWeakPassword if <8
func (s *Store) Verify(username, password string) error
type Sessions struct{ TTL time.Duration }               // NewSessions(ttl)
func NewSessions(ttl time.Duration) *Sessions
func (s *Sessions) Create() string
func (s *Sessions) Valid(token string) bool
func (s *Sessions) Delete(token string)
type Limiter struct{}                                   // NewLimiter(max int, window time.Duration, now func() time.Time)
func (l *Limiter) Allow() bool                          // false when failures in window >= max
func (l *Limiter) Fail()
const CookieName = "dd_session"
```

- [ ] Tests: setup then second setup → `ErrAlreadySetup`; short password → `ErrWeakPassword`; verify wrong password → `ErrBadCredentials`; session expires after TTL (inject clock via unexported field or short TTL); limiter blocks after 5 fails, allows after window passes.
- [ ] FAIL → implement → PASS. Commit.

### Task 3: `compose` package (runner, jobs, locks)

**Files:** Create `api/internal/compose/{runner.go,jobs.go,compose_test.go}`

**Interfaces — Produces:**
```go
type Runner interface {
    Run(ctx context.Context, dir string, out func(line string), args ...string) (exitCode int, err error)
}
type CLIRunner struct{}                                   // exec "docker compose ..." in dir, stdout+stderr merged, line by line
func ActionArgs(action string) ([][]string, bool)         // up:[[up -d --remove-orphans]], pull:[[pull],[up -d --remove-orphans]], stop, restart, down
func Validate(ctx context.Context, r Runner, dir, compose string) error // writes .dd-validate.yaml in dir, runs `-f <tmp> config -q`, removes tmp; error text = output
var ErrBusy error
type Job struct{ ID, Stack, Action string }
func (j *Job) Subscribe(ctx context.Context) <-chan Event // replays buffered lines then live, closes after Done
type Event struct{ Line string; Done bool; ExitCode int }
type Manager struct{}
func NewManager(r Runner, retention time.Duration) *Manager
func (m *Manager) Start(stack, dir, action string) (*Job, error) // ErrBusy if stack has running job
func (m *Manager) Get(id string) (*Job, bool)
func (m *Manager) RunSync(ctx context.Context, stack, dir string, args ...string) error // under same stack lock (used by delete)
```

- [ ] Tests (fake Runner that emits lines and blocks on a channel): late subscriber gets all lines + Done with exit code; second Start on same stack while running → `ErrBusy`, different stack OK; after completion lock released; pull runs two commands and stops on first nonzero exit; `ActionArgs("bogus")` → false.
- [ ] FAIL → implement → PASS (`go test -race`). Commit.

### Task 4: `docker` package

**Files:** Create `api/internal/docker/{docker.go,status_test.go}`

**Interfaces — Produces:**
```go
type Port struct{ Public, Private uint16; Proto string }
type Container struct{ ID, Name, Service, State, Status, Image string; Ports []Port }
type Client interface {
    ProjectContainers(ctx context.Context) (map[string][]Container, error) // keyed by com.docker.compose.project
    Logs(ctx context.Context, id string, tail int, out func(line string)) error // follow until ctx done
}
func NewClient() (Client, error)                       // client.FromEnv + API negotiation
func Status(cs []Container) string                     // running|partial|stopped|inactive
```

- [ ] Test `TestStatus` table: none→inactive, all running→running, mixed→partial, all exited→stopped. Implement Docker wrapper (logs demuxed with `stdcopy` when non-TTY). PASS. Commit.

### Task 5: `httpapi` + `cmd/server`

**Files:** Create `api/internal/httpapi/{server.go,auth_handlers.go,stack_handlers.go,sse.go,errors.go,server_test.go}`, `api/cmd/server/main.go`

**Interfaces — Consumes:** Tasks 1–4. **Produces:** `func New(deps Deps) http.Handler` with `Deps{Stacks *stacks.Store; Auth *auth.Store; Sessions *auth.Sessions; Limiter *auth.Limiter; Jobs *compose.Manager; Runner compose.Runner; Docker docker.Client; CookieSecure bool}`; endpoints exactly as the spec table.

- [ ] Tests via `httptest` with fake Runner + fake docker Client: unauthenticated `GET /api/stacks` → 401; setup→cookie set; second setup → 409; login wrong ×5 then 6th → 429; create bad name → 400; create with Validate failing → 422 with detail and no folder created; PUT invalid → 422 and file unchanged; DELETE never-deployed (runner returns exit 1 for down) → 204, folder gone; `POST /stacks/x/up` → 202 jobId; then `GET /jobs/{id}/stream` body contains `event: line` and `event: done`; busy → 409; `GET /stacks` returns status from fake docker.
- [ ] Implement; SSE helper sets `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`, flushes each event; data JSON-encoded string for `line`, `{"exitCode":n}` for `done`.
- [ ] `main.go`: env config, `MkdirAll` stacks/data dirs, graceful shutdown on SIGINT/SIGTERM; WriteTimeout 0 (SSE).
- [ ] `go vet ./... && go test -race ./...` → PASS. Commit.

### Task 6: API Dockerfile

**Files:** Create `api/Dockerfile`, `api/.dockerignore`
- [ ] Stage 1 `golang:1.26-alpine` → `CGO_ENABLED=0 go build -o /out/server ./cmd/server`. Stage 2 `docker:29-cli`, copy binary, `EXPOSE 8080`, `ENTRYPOINT ["/server"]`.
- [ ] `docker build -t docker-depoy-api ./api` succeeds. Commit.

### Task 7: Web scaffold, theme, API client

**Files:** Create `web/` via `npx shadcn@latest init --preset b1YmpVh4a --template next`; modify `web/app/globals.css`, `web/next.config.ts`; create `web/lib/api.ts`, `web/lib/sse.ts`, `web/lib/sse.test.ts`, `web/vitest.config.ts`

**Interfaces — Produces:**
```ts
// lib/api.ts
type StackStatus = 'running'|'partial'|'stopped'|'inactive'
type StackSummary = { name: string; status: StackStatus; containers: { running: number; total: number } }
type StackDetail = { name: string; compose: string; env: string; status: StackStatus }
type ContainerInfo = { id; name; service; state; status; image: string; ports: {public:number;private:number;proto:string}[] }
class ApiError extends Error { status: number; detail?: string }
const api = { authStatus, setup, login, logout, listStacks, getStack, createStack, saveStack, deleteStack, action(name, action): Promise<{jobId:string}>, containers(name) }
// lib/sse.ts
function parseSSE(chunk: string, carry: string): { events: {event:string; data:string}[]; carry: string }
function streamSSE(url: string, onEvent: (e:{event:string;data:string})=>void, signal: AbortSignal): Promise<void> // fetch + reader
```
- [ ] Theme: replace `:root`/`.dark` in globals.css with spec tokens. `next.config.ts`: `output: 'standalone'`, rewrites `/api/:path*` → `${process.env.API_URL ?? 'http://localhost:8080'}/api/:path*`, `compress: false`.
- [ ] Vitest `parseSSE` tests: split across chunks, multiple events per chunk, default event `message`.
- [ ] Install deps; add shadcn components: sidebar button input label card badge tabs sheet alert-dialog dropdown-menu command dialog skeleton sonner tooltip separator scroll-area textarea collapsible. `npm test` PASS. Commit.

### Task 8: Providers, auth pages, app shell

**Files:** Create `web/components/providers.tsx` (QueryClient, ThemeProvider, Toaster), `web/app/(auth)/login/page.tsx`, `web/app/(auth)/setup/page.tsx`, `web/app/page.tsx` (redirect by authStatus), `web/app/stacks/layout.tsx` (auth guard + SidebarProvider), `web/components/app-sidebar.tsx`, `web/components/status-dot.tsx`, `web/components/theme-toggle.tsx`, `web/components/stack-command.tsx`
- [ ] Sidebar per spec (brand, New stack, filter, list w/ dots polling 3s, footer theme + logout); Ctrl+K command palette. Any `ApiError` 401 → `router.replace('/login')`.
- [ ] `npm run build` PASS. Commit.

### Task 9: Stack pages

**Files:** Create `web/app/stacks/page.tsx` (empty state), `web/app/stacks/new/page.tsx`, `web/app/stacks/[name]/page.tsx`, `web/components/compose-editor.tsx`, `web/components/action-sheet.tsx`, `web/components/containers-tab.tsx`, `web/components/logs-view.tsx`, `web/components/delete-dialog.tsx`
- [ ] Detail header with actions (Deploy=up, Stop, Restart, Update=pull, menu Down/Delete), tabs Compose/Containers/Logs, Ctrl+S save, Save & Deploy, dirty tracking; ActionSheet streams `/api/jobs/{id}/stream`, toasts result, invalidates queries; Delete requires typing name; port links `http://${location.hostname}:${public}`.
- [ ] `npm run build && npm run lint && npm test` PASS. Commit.

### Task 10: Packaging + end-to-end

**Files:** Create `web/Dockerfile`, `web/.dockerignore`, `compose.yaml`, update `README.md`
- [ ] web Dockerfile: `node:24-alpine` deps → build → runtime copying `.next/standalone`, `.next/static`, `public`; `CMD ["node","server.js"]`, `ENV PORT=3000 HOSTNAME=0.0.0.0`.
- [ ] Root compose per spec (api: sock, `/opt/stacks:/opt/stacks`, `dd-data:/data`; web: `API_URL`, `5001:3000`, depends_on api).
- [ ] E2E: run API locally against Docker with temp STACKS_DIR + `npm run dev`; setup, create `hello` stack (nginx:alpine), deploy, see running + containers + logs, stop, delete. Commit.
