# Hostbox Onboarding

This guide is **implementation-first**. It describes what is in the repository today, where the important code lives, how the pieces fit together, and where the current gaps are.

It is intentionally more honest than aspirational:

- some parts of the backend are fairly complete
- some parts of the frontend and API have drifted apart
- automated checks exist, but end-to-end behavior is **not yet proven**

If you are starting to contribute, read this before trusting `SPEC.md` or the phase plans in `docs/`.

---

## 1. What Hostbox is today

Hostbox is a self-hosted deployment platform aimed at static/frontend apps. The main runtime is:

1. **Hostbox API**: a Go/Echo server
2. **SQLite**: embedded app database
3. **Docker**: used to build user projects in isolated containers
4. **Caddy**: separate reverse proxy and static file server
5. **React dashboard**: frontend under `web/`
6. **CLI**: separate Go binary under `cmd/cli`

At a high level, the intended flow is:

1. user creates a project
2. Hostbox triggers a deployment
3. worker clones repo and builds it in Docker
4. artifact is copied to `deployments/`
5. Caddy is updated to serve that artifact on preview/production URLs

That flow exists in code, but some surrounding pieces are incomplete or mismatched.

---

## 2. Current confidence level

This repo is **not yet a trusted end-to-end product**.

What currently looks strong:

- Go backend structure
- SQLite migrations and repositories
- worker/build pipeline shape
- Caddy config generation and sync services
- GitHub webhook plumbing

What currently lowers confidence:

- no browser end-to-end tests
- no verified VM/public-domain smoke test in CI
- some documented features are still TODOs or stubs

From the current checkout:

- `go test ./...` passes
- Docker images build
- frontend lint passes with warnings only
- frontend build succeeds locally and in Docker

Treat the project as **implemented enough to run locally and contribute to**, but **not yet validated enough to assume every hosted workflow is production-proven**.

---

## 3. Suggested reading order

Read these files in roughly this order:

1. `cmd/api/main.go`
2. `internal/config/config.go`
3. `internal/api/server.go`
4. `internal/api/routes/routes.go`
5. `internal/repository/`
6. `internal/services/auth.go`
7. `internal/services/deployment/service.go`
8. `internal/worker/executor.go`
9. `internal/services/caddy/`
10. `internal/services/github/`
11. `web/src/app.tsx`
12. `web/src/hooks/`
13. `cmd/cli/cmd/`

If you want the shortest “mental map”, start with:

- `cmd/api/main.go`
- `internal/worker/executor.go`
- `internal/services/caddy/builder.go`
- `web/src/app.tsx`

---

## 4. Repository map

| Path | Purpose |
| --- | --- |
| `cmd/api` | API server entrypoint |
| `cmd/cli` | CLI entrypoint and commands |
| `internal/api` | Echo server setup, middleware, handlers, routes |
| `internal/config` | env-driven config loading and validation |
| `internal/database` | SQLite open/migrate helpers |
| `internal/repository` | DB access layer |
| `internal/services` | auth, deployment, GitHub, Caddy, backup, scheduler, notifications |
| `internal/worker` | build executor, worker pool, SSE hub |
| `internal/platform/detect` | framework and package-manager detection |
| `internal/platform/docker` | Docker wrapper used by builds |
| `migrations` | embedded SQL schema migrations |
| `web` | React + Vite dashboard |
| `docker` | Dockerfiles for app and Caddy |
| `docker-compose*.yml` | dev/prod composition |
| `docs` | architecture/spec/plans; useful, but not always aligned with code |

---

## 5. Runtime architecture

### API startup flow

`cmd/api/main.go` wires the server in this order:

1. load config from environment
2. setup logger
3. open SQLite
4. run migrations
5. create repositories
6. create auth/backup/notification/update services
7. initialize Docker client
8. initialize Caddy client + config builder + sync service
9. optionally initialize GitHub App integration
10. create Echo server
11. create handlers
12. initialize build executor + worker pool if Docker is available
13. start background schedulers
14. register routes
15. start HTTP server and graceful shutdown hooks

Important consequence: **Docker is optional at startup**, but without it the build pipeline is effectively disabled.

### Database

Hostbox uses **SQLite in WAL mode** via `internal/database/sqlite.go`.

Migrations are embedded from `migrations/*.sql` and applied on startup. Core tables include:

- `users`
- `sessions`
- `projects`
- `deployments`
- `domains`
- `env_vars`
- `notification_configs`
- `activity_log`
- `settings`

The repositories under `internal/repository/` are the main source of truth for data access.

### HTTP server

`internal/api/server.go` creates the Echo server and attaches:

- request ID middleware
- structured request logging
- panic recovery
- CORS
- security headers
- custom JSON error handling

Route registration happens in `internal/api/routes/routes.go`.

### Build pipeline

The main build logic lives in `internal/worker/executor.go`.

The current flow is:

1. fetch deployment + project
2. create a build log file and SSE stream
3. mark deployment as `building`
4. clone repo
5. detect framework / package manager / Node version
6. create Docker build container with resource limits
7. run install + build commands
8. copy build output into the deployment artifact directory
9. mark deployment `ready`
10. run post-build hooks (Caddy + notifications)

The worker pool in `internal/worker/pool.go` handles queueing, panic recovery, crash recovery, and orphaned container cleanup.

### Caddy integration

`internal/services/caddy/` is responsible for generating and pushing Caddy config through the admin API.

Important files:

- `builder.go`: builds full Caddy JSON config
- `client.go`: talks to Caddy admin API
- `sync.go`: full sync on startup and periodic sync
- `manager.go`: route-level updates for individual deployments/domains

Caddy serves:

- the platform hostname
- production project routes
- branch-stable routes
- preview deployment routes
- verified custom domains

### GitHub integration

`internal/services/github/` contains:

- App auth/token management
- webhook event routing
- push handling
- pull request handling
- installation handling

This part is optional and only initializes when GitHub App config is present.

---

## 6. Backend API surfaces

Main route groups from `internal/api/routes/routes.go`:

### Public

- `GET /api/v1/health`
- `GET /api/v1/setup/status`
- `POST /api/v1/setup`
- auth routes: register, login, refresh, forgot-password, reset-password, verify-email

### Authenticated

- `/projects`
- `/deployments`
- `/domains`
- `/env-vars`
- `/auth/me`

### Admin

- `/api/v1/admin/stats`
- `/api/v1/admin/activity`
- `/api/v1/admin/users`
- `/api/v1/admin/settings`
- `/api/v1/admin/backups`
- `/api/v1/admin/update/check`

### Optional GitHub

- `POST /api/v1/github/webhook`
- `GET /api/v1/github/installations`
- `GET /api/v1/github/repos`

For contributor work, the important handlers are:

- `internal/api/handlers/auth.go`
- `internal/api/handlers/setup.go`
- `internal/api/handlers/projects.go`
- `internal/api/handlers/deployments.go`
- `internal/api/handlers/domains.go`
- `internal/api/handlers/env_vars.go`
- `internal/api/handlers/admin.go`

---

## 7. Frontend architecture

The dashboard lives in `web/`.

### Main structure

- `web/src/app.tsx`: router and top-level providers
- `web/src/main.tsx`: React entrypoint
- `web/src/hooks/`: API-facing React Query hooks
- `web/src/stores/auth-store.ts`: Zustand auth store
- `web/src/components/`: UI and feature components
- `web/src/pages/`: route-level pages

### Frontend stack

- React 18
- React Router
- TanStack Query
- Zustand
- Tailwind
- Radix UI / shadcn-style components
- React Hook Form + Zod

### Auth model

The frontend keeps the access token in Zustand and tries to bootstrap auth by calling the refresh endpoint and then `/auth/me`.

Main files:

- `web/src/stores/auth-store.ts`
- `web/src/hooks/use-auth.ts`
- `web/src/components/shared/auth-guard.tsx`
- `web/src/components/shared/setup-guard.tsx`

### Important reality check

The dashboard is now much closer to the backend contract than it was originally:

- setup status and login routing line up
- project detail returns project + latest deployment + domains
- notifications have real API handlers
- deployment rollback/redeploy actions now hit the correct routes
- log streaming works with the current SSE protocol

That said, the UI still has not been proven with automated browser tests.

---

## 8. CLI architecture

The CLI is a separate Go app under `cmd/cli`.

Key pieces:

- `cmd/cli/cmd/root.go`: root command and global flags
- `cmd/cli/internal/client/client.go`: HTTP client
- `cmd/cli/internal/config/config.go`: local CLI config/token storage
- `cmd/cli/internal/link/`: `.hostbox.json` project link handling

Implemented command groups include:

- `login`, `logout`, `whoami`
- `projects`, `project create`
- `link`
- `deploy`, `status`, `logs`, `rollback`
- `domains`
- `env`
- `admin`

The CLI is useful for understanding the intended API usage, but it is not feature-complete. For example, `hostbox logs` explicitly says real-time SSE log streaming is not fully wired for the CLI.

---

## 9. Local development

You have two realistic ways to work on the codebase.

Before either option, create a local env file:

```bash
cp .env.example .env
```

For a local-only setup, these values are enough:

```dotenv
PLATFORM_DOMAIN=localhost
PLATFORM_HTTPS=false
JWT_SECRET=dev-secret-dev-secret-dev-secret-1234
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

### Option A: run services directly

#### Backend

```bash
go mod download
set -a
source .env
set +a
CGO_ENABLED=1 go run ./cmd/api
```

#### Frontend

```bash
cd web
npm install
npm run dev
```

Then open:

- dashboard dev UI: `http://localhost:3000`
- API: `http://localhost:8080/api/v1/health`

On a fresh database the UI should send you to `/setup`. After setup you should be able to:

1. create the first admin account
2. sign in
3. create a project
4. trigger a deployment
5. watch build logs in the deployment detail page

### Important note about environment variables

The Go app reads directly from the process environment in `internal/config/config.go`. It does **not** load `.env` automatically.

That means:

- `.env` is useful for Docker Compose
- `go run ./cmd/api` needs exported env vars in your shell

At minimum you need valid values for:

- `JWT_SECRET`
- `ENCRYPTION_KEY`
- `PLATFORM_DOMAIN`

For purely local API work, something like this is enough:

```bash
export JWT_SECRET="dev-secret-dev-secret-dev-secret-1234"
export ENCRYPTION_KEY="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
export PLATFORM_DOMAIN="localhost"
export PLATFORM_HTTPS="false"
export DATABASE_PATH="./data/hostbox.db"
export LOG_LEVEL="debug"
export LOG_FORMAT="text"
```

### Option B: Docker Compose

#### Dev compose

```bash
docker compose -f docker-compose.dev.yml up
```

This intends to start:

- Go API on `:8080`
- Caddy on `:80/:443`
- frontend dev server in a Node container

The dev compose stack now aligns with the Vite dev server on port `3000`, so the expected local endpoints are:

- frontend: `http://localhost:3000`
- API: `http://localhost:8080`
- Caddy: `http://localhost`

Compose still depends on a valid `.env` file in the repo root for secrets like `JWT_SECRET` and `ENCRYPTION_KEY`.

### Local GitHub testing

The current GitHub flow is **GitHub App based**, not “sign in with GitHub OAuth”.

If you configure `GITHUB_APP_ID`, `GITHUB_APP_SLUG`, `GITHUB_APP_PEM`, and `GITHUB_WEBHOOK_SECRET`:

1. the dashboard can list app installations
2. the project creation wizard can list repos for an installation
3. Hostbox stores `github_installation_id` with the project
4. builds can clone installation-backed repos using an installation token

Practical limits:

- for **manual deployments**, a local setup is enough
- for **push-triggered deployments**, GitHub still needs a public webhook target, so use a tunnel such as ngrok or Cloudflare Tunnel
- for **custom domains**, you still need real DNS pointing at your instance

---

## 10. Production packaging

Production runtime is defined by:

- `docker/Dockerfile`
- `docker-compose.yml`
- `docker/caddy/Dockerfile`

### What the Dockerfile does

1. build the frontend in `web/dist`
2. build the Go API binary
3. build the Go CLI binary
4. package the binaries into an Alpine runtime image

### Current packaging behavior

The production image now includes `web/dist`, and the Go server serves the built dashboard for non-API routes. That means the platform hostname can be reverse proxied to the API process and still render the dashboard.

It is still filesystem-based serving, not an embedded `go:embed` bundle.

---

## 11. Testing and validation

### Existing automated checks

GitHub Actions in `.github/workflows/test.yml` runs:

- `go vet ./...`
- `go test -v -race ./...`
- Go builds for API and CLI
- frontend lint
- frontend type-check
- frontend build
- Docker image builds

### What is missing

- no browser end-to-end tests
- no deployment smoke tests against a running VM
- no verified “push repo, get working preview URL” test flow in the repo
- no contract tests between frontend and backend

### Practical takeaway

You should trust:

- unit-level backend behavior more than UI behavior
- Docker image buildability more than full platform usability

You should not assume:

- webhook-triggered deploys are proven locally without a tunnel
- custom domains can be tested without real DNS
- every dashboard flow has browser-level test coverage

---

## 12. Known implementation gaps

These are the main things a contributor should know before diving in.

### 1. No end-to-end browser coverage

The major dashboard flows now line up with the API more closely, but there is still no Playwright/Cypress-style suite proving:

- initial setup
- login refresh bootstrap
- project creation
- deployment logs
- notifications CRUD
- admin flows

Manual testing is still required for confidence.

### 2. Setup still relies on refresh-cookie behavior

The first-run setup flow now reaches the dashboard correctly in local testing, but it still deserves special scrutiny because setup/login/bootstrap auth each touch different auth paths.

If you change auth or setup code, re-test the first-run browser flow manually.

### 3. Domain verification is basic

Domain verification is no longer a stub, but the current check is still simple:

- it verifies that the hostname resolves in DNS
- it does **not** deeply validate that the record points to the expected Hostbox/Caddy target

That is good enough for a first local/manual implementation, but not strong enough to call finished.

### 5. Email flows are not complete

`internal/services/auth.go` contains TODO behavior for password reset email sending. Reset token generation exists, but SMTP-driven delivery is not really finished.

### 6. Some deployment follow-up flows are partial

`internal/services/deployment/service.go` still contains TODOs around route updates for rollback/promote flows.

The build pipeline and post-build hook integration are present, but not every deployment lifecycle path is fully wired.

### 6. Rollback/promote lifecycle still deserves scrutiny

`internal/services/deployment/service.go` still contains TODO notes around route updates for rollback/promote flows.

The core deployment path is much healthier now, but those follow-up lifecycle transitions still deserve targeted manual testing.

---

## 13. How to contribute effectively right now

If you want to make meaningful progress quickly, the best first contribution areas are:

### A. Add end-to-end confidence

The highest-value next step is no longer basic contract repair. It is proving the repaired flows with:

- browser tests for setup/login/project creation
- deployment smoke tests against a running local stack
- a documented webhook-tunnel test recipe

### B. Keep hardening setup/auth onboarding

Focus files:

- `internal/api/handlers/setup.go`
- `internal/services/auth.go`
- `web/src/hooks/use-setup-status.ts`
- `web/src/components/shared/setup-guard.tsx`

### C. Strengthen domain verification

The current verification path is intentionally simple and would benefit from:

- expected-target validation
- clearer DNS instructions for apex vs subdomain records
- tests around verified/unverified state changes

---

## 14. Debugging tips

### Useful backend entrypoints

- `cmd/api/main.go`
- `internal/api/routes/routes.go`
- `internal/api/handlers/deployments.go`
- `internal/worker/executor.go`

### Useful frontend entrypoints

- `web/src/app.tsx`
- `web/src/hooks/use-auth.ts`
- `web/src/hooks/use-projects.ts`
- `web/src/hooks/use-deployments.ts`

### Useful operational files

- `docker-compose.dev.yml`
- `docker-compose.yml`
- `.github/workflows/test.yml`
- `.env.example`

### Helpful API health check

```bash
curl http://localhost:8080/api/v1/health
```

### Helpful test command

```bash
CGO_ENABLED=1 go test ./... -count=1
```

---

## 15. Bottom line

Hostbox is **not just a spec**. There is real backend, worker, Docker, Caddy, GitHub, frontend, and CLI code here.

But it is also **not yet a coherent, battle-tested product**. The main thing to understand as a contributor is:

- the backend architecture is fairly real
- the dashboard exists, but has contract drift
- some “it should work like Vercel” surfaces are not fully wired
- the fastest path to making the project usable is reducing the gap between code paths that already exist

If you contribute with that mindset, you will make much faster progress than if you assume the docs already describe a working system.
