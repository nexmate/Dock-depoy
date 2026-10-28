# Docker-depoy — Design

A small, self-hosted, single-user web app for managing Docker Compose stacks on one host, in the spirit of Dockge. Go backend, Next.js + shadcn/ui frontend, modern look.

## Intent

- **Who:** one person managing their own server / homelab.
- **Outcome:** create, edit, deploy and observe Compose stacks from a browser instead of SSH.
- **Success:** stacks folder on disk is the source of truth (existing Dockge stacks just appear); every action behaves exactly like running `docker compose` by hand; the UI feels fast and modern in light and dark mode.
- **Out of scope for v1:** multi-user/roles, multiple hosts/agents, web terminal (exec), `docker run` converter, resource stats, dashboard/prune.

## Architecture

```
Browser ──► web: Next.js (:3000, only published port)
              │  /api/*  → rewrite proxy →  api: Go (:8080, internal network only)
              │                                ├─ `docker compose` CLI (stack actions)
              │                                ├─ Docker Engine SDK (containers, logs)
              │                                └─ $STACKS_DIR/<name>/compose.yaml (+ .env)
```

- Two services: `web` (Next.js standalone server) and `api` (Go). Only `web` is published.
- Browser talks to one origin; Next rewrites proxy `/api/*` to `API_URL` (default `http://api:8080`). No CORS; session cookie works as-is.
- Streaming (action output, container logs) uses **Server-Sent Events**, which pass through the rewrite proxy as plain HTTP.
- Go owns all logic. Next.js is UI + routing only.

## Go API (`api/`)

Go module `github.com/nenad/docker-depoy/api`, stdlib `net/http` routing, no framework.

```
cmd/server/main.go   config (env), wiring, graceful shutdown
internal/auth/       admin credential store, sessions, middleware, login rate limit
internal/stacks/     stack files on disk: list/read/write/delete, name validation
internal/compose/    Runner interface; CLI runner streams combined output; per-stack lock; job registry
internal/docker/     Engine SDK: containers by compose project label, status, log stream
internal/httpapi/    handlers, JSON + SSE helpers, error mapping
```

### Config (env)

| Var | Default | Meaning |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | API listen address |
| `STACKS_DIR` | `/opt/stacks` | Stacks root |
| `DATA_DIR` | `/data` | App data (`auth.json`) |
| `COOKIE_SECURE` | `false` | Set `Secure` on session cookie |

### Endpoints (prefix `/api`)

All require a session except `auth/status`, `auth/setup`, `auth/login`.

| Method | Path | Behaviour |
|---|---|---|
| GET | `/auth/status` | `{setupDone, authenticated}` |
| POST | `/auth/setup` | `{username, password}`; only when no admin exists (else 409); password ≥ 8 chars; logs in |
| POST | `/auth/login` | `{username, password}` → sets cookie; 401 on bad creds; 429 after 5 failures/min |
| POST | `/auth/logout` | clears session |
| GET | `/stacks` | `[{name, status, containers: {running, total}}]` sorted by name |
| POST | `/stacks` | `{name, compose, env}` → 201; 409 if exists; 400 bad name; 422 invalid compose |
| GET | `/stacks/{name}` | `{name, compose, env, status}`; 404 if missing |
| PUT | `/stacks/{name}` | `{compose, env}` → validates then writes; 422 invalid compose |
| DELETE | `/stacks/{name}` | runs `down` (ignore errors if never deployed), removes folder |
| POST | `/stacks/{name}/{action}` | action ∈ `up, stop, restart, pull, down` → `202 {jobId}`; 409 if stack busy |
| GET | `/jobs/{id}/stream` | SSE: `event: line` per output line, final `event: done` with `{exitCode}` |
| GET | `/stacks/{name}/containers` | `[{id, name, service, state, status, image, ports:[{public, private, proto}]}]` |
| GET | `/containers/{id}/logs?tail=200` | SSE `event: line` with follow; ends when client disconnects |

Errors: JSON `{error, detail?}` with 400/401/404/409/422/429/500.

### Compose actions

`compose.Runner.Run(ctx, dir, args...)` executes `docker compose <args>` with working directory = stack dir, streaming combined stdout+stderr lines.

| Action | Args |
|---|---|
| up | `up -d --remove-orphans` |
| stop | `stop` |
| restart | `restart` |
| pull | `pull` then `up -d --remove-orphans` |
| down | `down` |
| validate | `config -q` (used on create/save; compose file written to temp file in stack dir) |

Jobs: in-memory registry; each job buffers its lines so a late SSE subscriber replays from the start; jobs are evicted 10 minutes after finishing. A per-stack mutex rejects a second concurrent action with 409.

### Stack rules

- Name: `^[a-z0-9][a-z0-9_-]*$`, max 64 chars.
- Compose file name on read: first existing of `compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml`; written as the one found, else `compose.yaml`.
- `.env` optional; empty env on save deletes the file.
- Status from containers labelled `com.docker.compose.project=<name>`: all running → `running`; some running → `partial`; containers exist, none running → `stopped`; no containers → `inactive`.

### Auth

- `DATA_DIR/auth.json`: `{username, passwordHash}` (bcrypt).
- Sessions: 32-byte random token, in memory, 7-day expiry; cookie `dd_session`, `HttpOnly`, `SameSite=Strict`, `Path=/`.
- Rate limit: 5 failed logins per minute, global (every request arrives via the `web` proxy, so client IPs are not meaningful; single user anyway).

## Frontend (`web/`)

Scaffolded with `npx shadcn@latest init --preset b1YmpVh4a --template next`. Next.js App Router, `output: 'standalone'`. Libraries: TanStack Query, next-themes, `@uiw/react-codemirror` + `@codemirror/lang-yaml`, lucide-react, sonner.

### Theme

Neutral shadcn palette (user-provided tokens, light + dark, `--radius: 0.625rem`), stored verbatim in `globals.css`. Blue (`--chart-*` / dark `--sidebar-primary`) is the single accent, used for active nav and running state. Geist Sans for UI, Geist Mono for editor and logs.

### Layout & routes

shadcn `Sidebar` master/detail:

- Sidebar: brand, "New stack" button, filter input, stack list with status dots (green running, amber partial, grey stopped, outline inactive), footer with theme toggle and logout. Collapses to a drawer on mobile.
- `/setup`, `/login`: centred card forms. `/` redirects based on `auth/status`.
- `/stacks` (index): empty state.
- `/stacks/new`: name + compose editor prefilled with a starter template, `.env` editor; Create / Create & Deploy.
- `/stacks/[name]`: header (name, status badge, Deploy, Stop, Restart, Update (pull), overflow menu with Down / Delete); tabs:
  - **Compose:** YAML editor + collapsible `.env` editor; Save / Save & Deploy, enabled when dirty; `Ctrl+S` saves.
  - **Containers:** cards with state badge, image, port links (`http://<window host>:<public port>`), "Logs" button.
  - **Logs:** live tail of a selected container (defaults to first).
- Actions open a bottom Sheet streaming job output in a terminal panel, ending with a success/failure badge + toast. Down and Delete confirm via AlertDialog; Delete requires typing the stack name.
- `Ctrl+K` opens a command palette to jump to a stack.
- Polling: stack list and containers every 3 s.
- Auth guard: client-side; any 401 → `/login`; layout checks `auth/status` on load.

Next.js `rewrites` proxy `/api/:path*` → `${API_URL}/api/:path*`.

## Deployment

Root `compose.yaml`:

- `api`: built from `api/Dockerfile` — multi-stage, `golang` build → `docker:cli` runtime (ships `docker` + compose plugin). Mounts `/var/run/docker.sock`, `/opt/stacks:/opt/stacks` (same path inside and out, so relative bind mounts in stacks resolve on the host), and a named volume at `/data`. Not published.
- `web`: built from `web/Dockerfile` — Node build → standalone runtime. `API_URL=http://api:8080`. Publishes `5001:3000`.

## Testing

- Go: unit tests for `stacks` (temp dir), `auth` (store, sessions, rate limit), `compose` job registry/locking, and `httpapi` handlers against fake Runner + fake Docker client.
- Web: Vitest for the SSE stream parser and API client; `next build` + lint must pass.
- Manual: deploy a sample stack end-to-end against the local Docker host.
