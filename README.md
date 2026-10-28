Docker-depoy

A small, self-hosted web app for managing Docker Compose stacks on one host, in the spirit of Dockge.
Go API + Next.js / shadcn UI.

- Stacks are folders in `/opt/stacks/<name>/` with a `compose.yaml` (and optional `.env`); existing Dockge stacks just appear.
- Create and edit compose files with validation (`docker compose config`).
- Deploy, stop, restart, pull & redeploy, down, delete — with live output.
- Container list per stack with port links, and live container logs.
- Single admin account, light / dark mode, `Ctrl+K` stack search, `Ctrl+S` save.

## Run

```bash
docker compose up -d --build
```

Open http://localhost:5001 and create the admin account. Set `DD_PORT` to publish on another port.

The `api` container mounts the Docker socket and `/opt/stacks` at the same path as on the host, so relative
bind mounts inside your stacks resolve correctly. Credentials live in the `dd-data` volume.

## Develop

```bash
cd api && STACKS_DIR=./stacks DATA_DIR=./data go run ./cmd/server
```

```bash
cd web && npm install && npm run dev
```

The dev UI on http://localhost:3000 proxies `/api` to `http://localhost:8080` (override with `API_URL`).

Tests: `cd api && go test ./...` and `cd web && npm test`.

## Layout

- `api/` — Go 1.26, stdlib `net/http`. `internal/stacks` (files), `internal/compose` (CLI runner + jobs),
  `internal/docker` (Engine SDK), `internal/auth`, `internal/httpapi` (REST + SSE).
- `web/` — Next.js 16 App Router, shadcn/ui (Base UI), TanStack Query, CodeMirror.
- `docs/superpowers/` — design spec and implementation plan.
