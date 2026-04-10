# Hostbox - Self-Hostable Vercel Alternative

> Deploy frontend applications with a Vercel-like developer experience on your own VPS.

**Version:** v1.0.0
**Status:** Planning

---

## 1. Overview

### 1.1 Vision

Hostbox is a self-hostable deployment platform that brings Vercel's magical developer experience to anyone with a VPS. It's designed for developers who want the simplicity of Vercel but need full control over their infrastructure.

**The key differentiator from Coolify/Dokploy**: Hostbox is radically lightweight. It runs on a $4/month VPS (512MB RAM, 10GB disk) because it focuses exclusively on frontend/static/JAMstack deployments and avoids the bloat of general-purpose PaaS platforms.

### 1.2 Target Users

- **Solo developers** who want easy deployments without cloud vendor complexity
- **Small teams** needing a private deployment platform
- **DevOps beginners** learning deployment workflows
- **Privacy-conscious developers** who want data on their own servers
- **Cost-conscious developers** running on cheap VPS ($4-6/month tier)

### 1.3 v1 Goals

- Zero-config GitHub integration with auto-framework detection
- Automatic HTTPS with Let's Encrypt
- Preview deployments for every PR
- Custom domains with SSL
- One-click rollbacks
- Real-time build logs
- One-line install script (`curl -sSL https://hostbox.sh/install | bash`)
- Run on 512MB RAM / 10GB disk VPS

### 1.4 Non-Goals (v1)

- Multiple build workers (single server builds)
- GitLab/Bitbucket support (GitHub-only in v1)
- Kubernetes or Docker Swarm deployment
- Database hosting (PostgreSQL, MySQL, Redis, etc.)
- Docker Compose / arbitrary container deployments
- Server-side rendering runtime (SSR; only static export / SSG)
- Multi-server deployment
- Template marketplace

### 1.5 Design Principles

1. **Minimal footprint** — Two processes (Go binary + Caddy). No PostgreSQL, no Redis, no Node.js runtime.
2. **Single binary** — API server, build worker, and embedded web dashboard compile into one Go binary.
3. **Convention over configuration** — Auto-detect framework, build command, and output directory.
4. **Fail gracefully** — Every component should degrade gracefully (no SMTP? skip email verification. No GitHub App? allow manual deploys).

---

## 2. Architecture

### 2.1 System Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                          User's VPS                                │
│                                                                    │
│  ┌──────────────────────────────────┐       ┌──────────┐           │
│  │         hostbox (Go binary)      │       │  Caddy   │           │
│  │                                  │       │  :80/443 │           │
│  │  ┌──────────┐  ┌──────────────┐  │       │          │           │
│  │  │ API      │  │ Build Worker │  │◄─────►│ Reverse  │           │
│  │  │ Server   │  │ (goroutines) │  │ Admin │ Proxy +  │           │
│  │  ├──────────┤  ├──────────────┤  │  API  │ Auto SSL │           │
│  │  │ Embedded │  │ Docker SDK   │  │       └──────────┘           │
│  │  │ Web UI   │  │ (build only) │  │              │               │
│  │  └──────────┘  └──────────────┘  │              │               │
│  │         │                        │        Static File           │
│  │    ┌────┴────┐                   │        Serving               │
│  │    │ SQLite  │                   │              │               │
│  │    │  (DB)   │                   │    /app/deployments/{id}/    │
│  │    └─────────┘                   │                              │
│  └──────────────────────────────────┘                              │
└────────────────────────────────────────────────────────────────────┘
         ↑                        ↑
    GitHub Webhooks          Users / Browsers
       GitHub App
```

**Why this architecture?**
- **SQLite instead of PostgreSQL**: Saves ~150MB RAM. For a single-server platform with <100 projects, SQLite is more than sufficient and eliminates an entire container.
- **Single binary instead of API + Worker + Web**: Saves ~200MB RAM. The build worker runs as goroutines within the same process, using a bounded worker pool. The web dashboard is a Vite React SPA embedded as static files.
- **Caddy as the only separate process**: Caddy is purpose-built for automatic HTTPS and static file serving. Its admin API enables hot-reload of routing config without restarts.

### 2.2 Components

| Component | Technology | Responsibility |
|-----------|------------|----------------|
| API Server | Go + Echo | REST API, webhook handling, orchestration |
| Build Worker | Go + Docker SDK (in-process) | Execute builds in isolated Docker containers |
| Web Dashboard | Vite + React (embedded in Go binary) | User interface, served as static files |
| CLI | Go + Cobra (separate binary) | Terminal workflow |
| Database | SQLite + WAL mode | Metadata storage, build logs |
| Reverse Proxy | Caddy | Routing, automatic HTTPS, static file serving |
| File Storage | Local disk | Build artifacts |

### 2.3 Memory Budget (Target: 512MB VPS)

| Component | Estimated RAM |
|-----------|--------------|
| Linux OS + Docker daemon | ~150MB |
| Hostbox binary (idle) | ~30MB |
| Caddy | ~20MB |
| SQLite (in-process) | ~5MB |
| Build container (during build) | ~256MB (temporary) |
| **Total (idle)** | **~200MB** |
| **Total (during build)** | **~450MB** |

### 2.4 Data Flow

```
[GitHub Push / PR]
      │
      ▼
[GitHub App Webhook]
      │
      ▼
[API Server receives event]
      ├── Verifies webhook signature (HMAC-SHA256)
      ├── Resolves project from repository + installation ID
      ├── Checks for existing queued/building deploy (deduplication)
      ├── Creates Deployment record (status: QUEUED)
      ├── Posts GitHub Deployment Status: "pending"
      └── Returns 202 Accepted
                │
                ▼
[Build Worker pool picks up job] (bounded: 1 concurrent build)
      ├── Updates status → BUILDING
      ├── Shallow-clones repository (depth=1, using GitHub App installation token)
      ├── Detects framework (package.json analysis)
      ├── Creates isolated Docker container with resource limits
      ├── Mounts build cache volume (node_modules, .next/cache, etc.)
      ├── Injects environment variables (scoped by preview/production)
      ├── Runs: install → build (streams logs via SSE)
      ├── Copies build output to /app/deployments/{project_id}/{deployment_id}/
      └── Cleans up container + cloned source
                │
                ▼
[Deployment Complete]
      ├── Updates status → READY (or FAILED with error message)
      ├── Posts GitHub Deployment Status: "success" / "failure"
      ├── Sends notification (Discord/Slack webhook, if configured)
      └── Updates Caddy routing via Admin API (hot reload, zero downtime)
                │
                ▼
[Caddy serves traffic]
      ├── {slug}-{short-id}.preview.{platform-domain} → artifact path
      └── custom-domain.com → artifact path (production)
```

---

## 3. Features

### 3.1 Authentication

**First-Run Setup**
- On first launch, Hostbox enters setup mode
- User creates an admin account (email + password)
- No external dependencies required (no SMTP needed for initial setup)
- Setup wizard configures platform domain, GitHub App, etc.

**Registration**
- Email + password (admin can invite additional users)
- Passwords hashed with bcrypt (cost factor 12)
- Email verification via SMTP (optional — if SMTP not configured, accounts are auto-verified)
- Admin can disable public registration (default: disabled after first user)

**Login**
- Email + password
- JWT tokens (access: 15min, refresh: 7 days)
- Refresh token stored in httpOnly cookie with `Secure`, `SameSite=Strict`

**Session Management**
- Multiple concurrent sessions allowed (browser + CLI + API)
- Per-session refresh tokens tracked in DB
- Logout invalidates specific session (or all sessions via "logout everywhere")
- Stale sessions auto-cleaned after refresh token expiry

**Password Recovery**
- `POST /auth/forgot-password` sends reset link (if SMTP configured)
- Reset tokens expire in 1 hour, single-use
- If no SMTP: admin can reset passwords via CLI (`hostbox admin reset-password <email>`)

### 3.2 Projects

**Create Project**
- Name (slug auto-generated: lowercase, hyphens, alphanumeric)
- GitHub repository selection (via GitHub App installation)
- Framework auto-detection from `package.json` (see §9.2)
- Override-able: build command, install command, output directory, root directory

**Project Settings**
- Change name/slug
- Update build configuration (commands, output dir, root dir, Node.js version)
- Configure production branch (default: `main`)
- Toggle auto-deploy on push (default: enabled)
- Toggle preview deployments on PR (default: enabled)
- Delete project (cascades: deployments, artifacts, domains, env vars)

**Project Limits (per instance)**
- Configurable max projects (default: 50)
- Configurable max deployments retained per project (default: 20)

### 3.3 Deployments

**Automatic Deployments**
- Push to production branch → Production deployment
- Push to PR branch / PR opened / PR synchronized → Preview deployment
- PR closed → Preview URL disabled (artifact retained per retention policy)
- Branch deleted → Associated preview deployments marked inactive

**Manual Deployments**
- Trigger deployment from dashboard or CLI
- Select branch and optionally a specific commit

**Deployment Deduplication**
- If a build is QUEUED or BUILDING for the same branch, the new push cancels the old one and starts fresh
- Prevents wasted resources on rapid pushes

**Deployment States**
```
QUEUED → BUILDING → READY
   ↓        ↓
CANCELLED  FAILED
```

**Rollback**
- One-click rollback to any previous successful deployment
- Rollback is instant — points Caddy to the existing artifact directory
- Creates new deployment record with `is_rollback=true` referencing the source deployment
- No rebuild required

**Deployment Metadata**
- Commit SHA, commit message, commit author
- Branch name
- Build duration
- Artifact size
- GitHub PR number (if preview)

### 3.4 Preview URLs

**Format**
```
{project-slug}-{short-hash}.{platform-domain}
```
Short hash = first 8 characters of deployment ID (nanoid).

**Examples**
```
my-app-a1b2c3d4.hostbox.example.com
dashboard-x9y8z7w6.hostbox.example.com
```

**Routing**
- Wildcard DNS: `*.{platform-domain}` pointed to VPS IP
- Caddy routes based on subdomain prefix
- Wildcard SSL via Let's Encrypt (DNS-01 challenge with supported providers, or HTTP-01 for non-wildcard)

**Branch-Stable Preview URLs**
- Each PR/branch also gets a stable URL that always points to the latest deployment:
  `{project-slug}-{branch-slug}.{platform-domain}`
- Example: `my-app-feat-login.hostbox.example.com`

### 3.5 Custom Domains

**Adding a Domain**
1. User enters domain name in dashboard or CLI
2. Platform displays DNS instructions:
   - **A record**: `@ → {server-ip}`
   - **CNAME**: `www → {project-slug}.{platform-domain}`
3. User configures DNS at their registrar
4. User clicks "Verify" — Hostbox checks DNS resolution
5. On successful verification, Caddy provisions SSL certificate automatically

**Domain Verification**
- DNS-based verification (check A/CNAME resolution to server IP)
- Periodic re-verification (every 24 hours) to detect stale domains
- Grace period of 7 days before removing unverifiable domains

**SSL**
- Let's Encrypt with automatic renewal via Caddy
- HTTP-01 challenge for individual custom domains
- Automatic renewal before expiry (Caddy handles this natively)
- Force HTTPS with automatic HTTP→HTTPS redirect

**Limits**
- Max 10 custom domains per project (v1)

### 3.6 Environment Variables

**Per Project**
- Key-value pairs
- Encrypted at rest (AES-256-GCM, encryption key derived from a master key in config)
- Secret values write-only in dashboard (can update or delete, never read back)

**Scopes**
- `production` — Injected only for production branch deployments
- `preview` — Injected only for preview deployments
- `all` — Injected for all deployments (default)

**Bulk Operations**
- Import from `.env` file (paste or upload)
- Export as `.env` file (non-secret values only)

**Built-in Variables** (auto-injected, cannot be overridden)
```
CI=true
HOSTBOX=true
HOSTBOX_URL=https://hostbox.example.com
HOSTBOX_DEPLOYMENT_ID=abc123
HOSTBOX_DEPLOYMENT_URL=https://my-app-abc123.hostbox.example.com
HOSTBOX_PROJECT_ID=xyz
HOSTBOX_PROJECT_NAME=my-app
HOSTBOX_BRANCH=feat/login
HOSTBOX_COMMIT_SHA=a1b2c3d4e5f6
HOSTBOX_COMMIT_REF=refs/heads/feat/login
HOSTBOX_IS_PREVIEW=true
NODE_ENV=production
```

### 3.7 Build Logs

**Real-time Streaming**
- SSE endpoint: `GET /api/v1/deployments/:id/logs/stream`
- Events: `log`, `status`, `error`, `complete`
- Supports reconnection via `Last-Event-ID` header

**Log Storage**
- Stored as plain text files on disk: `/app/logs/{deployment_id}.log`
- Indexed in SQLite (deployment_id → log file path, line count, size)
- NOT stored in the database (avoids SQLite bloat)

**Log Retention**
- Logs retained for the same duration as deployment artifacts
- Configurable retention: default 30 days or last N deployments per project (whichever is more)
- Old logs cleaned up by the garbage collector (§9.5)

### 3.8 Notifications

**Channels**
- Discord webhook
- Slack webhook (incoming webhook URL)
- Generic HTTP webhook (POST with JSON payload)

**Events**
- Deployment started
- Deployment succeeded (includes URL)
- Deployment failed (includes error summary)
- Domain verified / verification failed

**Configuration**
- Per-project notification settings
- Global default notification settings
- Notification payload includes: project name, branch, commit, status, URL, duration

### 3.9 System Dashboard

**Server Metrics** (lightweight, no Prometheus/Grafana)
- Disk usage: total, used, available, per-project breakdown
- Active deployments count
- Build queue status
- Uptime

**Activity Log**
- Recent deployments across all projects
- Domain changes
- Login events
- Searchable and filterable

---

## 4. Database Schema

### 4.1 Engine: SQLite with WAL Mode

SQLite in WAL (Write-Ahead Logging) mode provides:
- Concurrent reads during writes
- Crash-safe transactions
- Zero configuration / zero memory overhead
- Database is a single file: `/app/data/hostbox.db`

**Backup**: Simple file copy (or automated via `hostbox backup` CLI command). Optional integration with Litestream for continuous replication to S3-compatible storage.

### 4.2 ER Diagram

```
┌──────────┐       ┌─────────────┐
│   User   │──────<│   Session   │
└────┬─────┘       └─────────────┘
     │
     │ owner_id
     │
┌────┴──────┐
│  Project  │──────<┌─────────────┐
└─────┬─────┘       │   Domain    │
      │             └─────────────┘
      │
      ├────────────<┌─────────────┐
      │             │   EnvVar    │
      │             └─────────────┘
      │
      ├────────────<┌──────────────┐
      │             │  Deployment  │
      │             └──────────────┘
      │
      └────────────<┌──────────────────┐
                    │ NotificationCfg  │
                    └──────────────────┘
```

### 4.3 Tables

```sql
-- Users
CREATE TABLE users (
    id TEXT PRIMARY KEY,  -- nanoid
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name TEXT,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Sessions (refresh tokens)
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,  -- nanoid
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL,  -- SHA-256 of refresh token
    user_agent TEXT,
    ip_address TEXT,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Projects
CREATE TABLE projects (
    id TEXT PRIMARY KEY,  -- nanoid
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    github_repo TEXT,              -- 'owner/repo'
    github_installation_id INTEGER,
    production_branch TEXT NOT NULL DEFAULT 'main',
    framework TEXT,                -- detected: 'nextjs', 'vite', 'cra', 'astro', etc.
    build_command TEXT,            -- NULL = use framework default
    install_command TEXT,          -- NULL = use framework default
    output_directory TEXT,         -- NULL = use framework default
    root_directory TEXT DEFAULT '/',  -- for monorepos
    node_version TEXT DEFAULT '20',
    auto_deploy BOOLEAN NOT NULL DEFAULT TRUE,
    preview_deployments BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_projects_owner_id ON projects(owner_id);
CREATE INDEX idx_projects_github_repo ON projects(github_repo);

-- Deployments
CREATE TABLE deployments (
    id TEXT PRIMARY KEY,  -- nanoid
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    commit_sha TEXT NOT NULL,
    commit_message TEXT,
    commit_author TEXT,
    branch TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
        -- 'queued', 'building', 'ready', 'failed', 'cancelled'
    is_production BOOLEAN NOT NULL DEFAULT FALSE,
    deployment_url TEXT,              -- full URL once deployed
    artifact_path TEXT,               -- /app/deployments/{project_id}/{deployment_id}/
    artifact_size_bytes INTEGER,
    log_path TEXT,                    -- /app/logs/{deployment_id}.log
    error_message TEXT,               -- populated on failure
    is_rollback BOOLEAN NOT NULL DEFAULT FALSE,
    rollback_source_id TEXT REFERENCES deployments(id),
    github_pr_number INTEGER,
    build_duration_ms INTEGER,
    started_at TEXT,
    completed_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_deployments_project_id ON deployments(project_id);
CREATE INDEX idx_deployments_project_status ON deployments(project_id, status);
CREATE INDEX idx_deployments_project_branch ON deployments(project_id, branch);
CREATE INDEX idx_deployments_created_at ON deployments(created_at);

-- Custom domains
CREATE TABLE domains (
    id TEXT PRIMARY KEY,  -- nanoid
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    domain TEXT UNIQUE NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TEXT,
    last_checked_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_domains_project_id ON domains(project_id);

-- Environment variables
CREATE TABLE env_vars (
    id TEXT PRIMARY KEY,  -- nanoid
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    encrypted_value BLOB NOT NULL,  -- AES-256-GCM encrypted
    is_secret BOOLEAN NOT NULL DEFAULT FALSE,
    scope TEXT NOT NULL DEFAULT 'all',  -- 'all', 'preview', 'production'
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(project_id, key, scope)
);
CREATE INDEX idx_env_vars_project_id ON env_vars(project_id);

-- Notification configurations
CREATE TABLE notification_configs (
    id TEXT PRIMARY KEY,  -- nanoid
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,  -- NULL = global
    channel TEXT NOT NULL,       -- 'discord', 'slack', 'webhook'
    webhook_url TEXT NOT NULL,   -- encrypted
    events TEXT NOT NULL DEFAULT 'all',  -- comma-separated: 'deploy_success,deploy_failed'
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Activity log (lightweight audit trail)
CREATE TABLE activity_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,         -- 'deployment.created', 'project.deleted', 'domain.verified', etc.
    resource_type TEXT NOT NULL,  -- 'project', 'deployment', 'domain', etc.
    resource_id TEXT,
    metadata TEXT,               -- JSON blob with extra context
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_activity_log_created_at ON activity_log(created_at);
CREATE INDEX idx_activity_log_resource ON activity_log(resource_type, resource_id);

-- Platform settings (key-value store for system config)
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Pre-populated settings
INSERT INTO settings (key, value) VALUES
    ('setup_complete', 'false'),
    ('registration_enabled', 'false'),
    ('max_projects', '50'),
    ('max_deployments_per_project', '20'),
    ('artifact_retention_days', '30'),
    ('max_concurrent_builds', '1');
```

### 4.4 Notes on Schema Design

- **No teams table in v1**: Simplifies the model significantly. Projects belong to a user. Multi-user collaboration is handled via simple user accounts with per-project access (post-v1 feature).
- **nanoid for IDs**: Shorter than UUID, URL-safe, and collision-resistant. Generated in Go.
- **TEXT timestamps**: ISO 8601 strings work well with SQLite and Go's `time.Time`.
- **Logs on disk, not in DB**: Build logs can be hundreds of KB each. Storing them as files prevents SQLite bloat and allows streaming reads.
- **No SSL cert storage**: Caddy manages its own certificate storage. The platform never touches private keys.
- **No GitHub installations table**: The `github_installation_id` on the project is sufficient.

---

## 5. API Specification

### 5.1 Base URL

```
/api/v1
```

### 5.2 Authentication

All authenticated endpoints require:
```
Authorization: Bearer <access_token>
```

Refresh tokens are sent via httpOnly cookie:
```
Cookie: hostbox_refresh=<token>
```

**CORS**: The API sets `Access-Control-Allow-Origin` to the platform domain only. No wildcard origins.

**Rate Limiting**:
- Auth endpoints: 10 requests/minute per IP
- API endpoints: 100 requests/minute per user
- Webhook endpoint: 500 requests/minute (GitHub can burst)
- Rate limit headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

### 5.3 Endpoints

#### System

```
GET    /health
       Response: { status: "ok", version: "1.0.0", uptime_seconds: 12345 }

GET    /setup/status
       Response: { setup_complete: false }

POST   /setup
       Body: { email, password, platform_domain }
       Response: { user, access_token }
       Note: Only callable when setup_complete=false
```

#### Authentication

```
POST   /auth/register
       Body: { email, password, display_name? }
       Response: { user, access_token }
       Note: Only when registration is enabled

POST   /auth/login
       Body: { email, password }
       Response: { user, access_token }
       Sets Cookie: hostbox_refresh=<token>

POST   /auth/logout
       Response: { success: true }
       Clears Cookie: hostbox_refresh

POST   /auth/logout-all
       Response: { success: true, sessions_revoked: 3 }

POST   /auth/refresh
       Cookie: hostbox_refresh=<token>
       Response: { access_token }

POST   /auth/verify-email
       Body: { token }
       Response: { success: true }

POST   /auth/forgot-password
       Body: { email }
       Response: { success: true }
       Note: Always returns success (prevents email enumeration)

POST   /auth/reset-password
       Body: { token, new_password }
       Response: { success: true }

GET    /auth/me
       Response: { user }

PATCH  /auth/me
       Body: { display_name?, email?, current_password? }
       Response: { user }

PUT    /auth/me/password
       Body: { current_password, new_password }
       Response: { success: true }
```

#### Projects

```
GET    /projects
       Query: ?page=1&per_page=20&search=
       Response: { projects: [...], pagination: { total, page, per_page, total_pages } }

POST   /projects
       Body: { name, github_repo?, build_command?, install_command?,
               output_directory?, root_directory?, node_version? }
       Response: { project }

GET    /projects/:id
       Response: { project, latest_deployment?, domains }

PATCH  /projects/:id
       Body: { name?, build_command?, install_command?, output_directory?,
               root_directory?, node_version?, production_branch?,
               auto_deploy?, preview_deployments? }
       Response: { project }

DELETE /projects/:id
       Response: { success: true }
```

#### Deployments

```
GET    /projects/:projectId/deployments
       Query: ?page=1&per_page=20&status=&branch=
       Response: { deployments: [...], pagination }

POST   /projects/:projectId/deployments
       Body: { branch?, commit_sha? }
       Response: { deployment }
       Note: Triggers manual deployment

GET    /deployments/:id
       Response: { deployment }

POST   /deployments/:id/cancel
       Response: { deployment }
       Note: Only works for queued/building deployments

POST   /deployments/:id/rollback
       Response: { deployment }
       Note: Creates new deployment pointing to this one's artifacts

GET    /deployments/:id/logs
       Query: ?offset=0&limit=1000
       Response: { lines: [...], total_lines, has_more }

GET    /deployments/:id/logs/stream
       Response: SSE stream
       Events:
         event: log       data: { line: 1, message: "...", timestamp: "..." }
         event: status    data: { status: "building", phase: "Installing..." }
         event: error     data: { message: "Build failed: ..." }
         event: complete  data: { status: "ready", duration_ms: 67000, url: "..." }
       Headers: Last-Event-ID for reconnection
```

#### Domains

```
GET    /projects/:projectId/domains
       Response: { domains: [...] }

POST   /projects/:projectId/domains
       Body: { domain }
       Response: { domain, dns_instructions: { a_record, cname_record } }

DELETE /domains/:id
       Response: { success: true }

POST   /domains/:id/verify
       Response: { domain }
       Note: Checks DNS resolution and triggers SSL provisioning
```

#### Environment Variables

```
GET    /projects/:projectId/env-vars
       Response: { env_vars: [...] }
       Note: Secret values returned as "••••••••"

POST   /projects/:projectId/env-vars
       Body: { key, value, is_secret?, scope? }
       Response: { env_var }

POST   /projects/:projectId/env-vars/bulk
       Body: { env_vars: [{ key, value, is_secret?, scope? }, ...] }
       Response: { env_vars: [...], created: 5, updated: 2 }
       Note: Used for .env file import

PATCH  /env-vars/:id
       Body: { value?, is_secret?, scope? }
       Response: { env_var }

DELETE /env-vars/:id
       Response: { success: true }
```

#### GitHub

```
POST   /github/webhook
       Headers: X-Hub-Signature-256, X-GitHub-Event, X-GitHub-Delivery
       Body: GitHub webhook payload
       Response: 202 Accepted
       Note: Signature verified via HMAC-SHA256

GET    /github/repos
       Query: ?installation_id=&page=1&per_page=30
       Response: { repos: [...], pagination }

GET    /github/installations
       Response: { installations: [...] }
```

#### Notifications

```
GET    /projects/:projectId/notifications
       Response: { notifications: [...] }

POST   /projects/:projectId/notifications
       Body: { channel, webhook_url, events? }
       Response: { notification }

PATCH  /notifications/:id
       Body: { webhook_url?, events?, enabled? }
       Response: { notification }

DELETE /notifications/:id
       Response: { success: true }

POST   /notifications/:id/test
       Response: { success: true }
       Note: Sends a test notification
```

#### System (Admin only)

```
GET    /admin/stats
       Response: { disk_usage, project_count, deployment_count,
                   active_builds, uptime_seconds }

GET    /admin/activity
       Query: ?page=1&per_page=50&action=&resource_type=
       Response: { activities: [...], pagination }

GET    /admin/users
       Response: { users: [...] }

POST   /admin/settings
       Body: { registration_enabled?, max_projects?, max_concurrent_builds?,
               artifact_retention_days? }
       Response: { settings }
```

### 5.4 Error Responses

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input",
    "details": [
      { "field": "email", "message": "Invalid email format" }
    ]
  }
}
```

| Code | HTTP Status | Description |
|------|-------------|-------------|
| VALIDATION_ERROR | 400 | Invalid request body |
| UNAUTHORIZED | 401 | Missing or invalid token |
| FORBIDDEN | 403 | Insufficient permissions |
| NOT_FOUND | 404 | Resource doesn't exist |
| CONFLICT | 409 | Resource already exists |
| RATE_LIMITED | 429 | Too many requests |
| SETUP_REQUIRED | 503 | Platform not yet initialized |
| INTERNAL_ERROR | 500 | Server error |

### 5.5 Pagination Convention

All list endpoints use consistent pagination:
```json
{
  "data": [...],
  "pagination": {
    "total": 156,
    "page": 1,
    "per_page": 20,
    "total_pages": 8
  }
}
```

Query parameters: `?page=1&per_page=20` (max per_page: 100)

---

## 6. Web Dashboard

### 6.1 Technology

**Vite + React + TypeScript** compiled to static assets and embedded in the Go binary via `embed.FS`. This eliminates the need for a Node.js runtime in production.

**UI Framework**: Tailwind CSS + shadcn/ui for a modern, accessible design system.

### 6.2 Pages

```
/setup                       → First-run setup wizard (if not initialized)

/login                       → Login form
/register                    → Registration form (if enabled)
/forgot-password             → Password reset request
/reset-password              → Password reset form

/                            → Dashboard overview (all projects, recent deploys, server stats)

/projects/new                → Create project (GitHub repo picker + framework detection)
/projects/:slug              → Project detail (latest deploy, recent deploys, status)
/projects/:slug/deployments  → Deployment history (filterable by branch, status)
/projects/:slug/deployments/:id → Deployment detail + real-time log viewer
/projects/:slug/settings     → Project settings (build config, danger zone)
/projects/:slug/domains      → Domain management
/projects/:slug/env          → Environment variables
/projects/:slug/notifications → Notification webhooks

/settings                    → Account settings (profile, password, sessions)
/settings/admin              → Admin panel (users, system settings, activity log)
```

### 6.3 Key Components

**Layout**
- Responsive sidebar navigation (collapsible on mobile)
- Breadcrumb header with project context
- Dark mode support (system preference + manual toggle)
- Command palette (Cmd+K) for quick navigation

**Common Components**
- Button (variants: primary, secondary, destructive, ghost, outline)
- Input (text, password, select, textarea)
- Card, Badge, Modal, Toast notifications
- DataTable with pagination, sorting, and search
- EmptyState illustrations
- Skeleton loading states

**Feature Components**
- `DeploymentStatusBadge` — Color-coded status with animation for building
- `LogViewer` — Terminal-style viewer with ANSI color support, auto-scroll, search within logs
- `DomainCard` — DNS instructions, verification status, SSL status
- `EnvVarEditor` — Inline editing, show/hide secrets, bulk import from .env
- `BuildProgress` — Step indicator (Clone → Install → Build → Deploy)
- `CommitInfo` — SHA (truncated + copy), message, author avatar
- `FrameworkBadge` — Detected framework with icon (Next.js, Vite, Astro, etc.)
- `ServerStats` — Disk usage bar, active builds indicator
- `SetupWizard` — Multi-step onboarding flow

### 6.4 Real-time Updates

- **SSE (Server-Sent Events)** for build log streaming (not WebSocket — simpler, works through proxies)
- **Polling** for deployment status on project pages (every 5 seconds when a build is active)
- Auto-reconnection with exponential backoff on SSE disconnect

### 6.5 Onboarding Flow (First Run)

1. Welcome screen with overview
2. Create admin account
3. Configure platform domain
4. GitHub App creation guide (step-by-step with screenshots/links)
5. Verify GitHub webhook connectivity
6. Create first project
7. Dashboard

---

## 7. CLI

### 7.1 Distribution

The CLI is a separate Go binary (`hostbox-cli`) distributed as:
- Pre-built binaries (Linux amd64/arm64, macOS amd64/arm64)
- Install script: `curl -sSL https://hostbox.sh/cli | bash`
- Or built from source

### 7.2 Commands

```bash
# Authentication
hostbox login                     # Interactive login (opens browser or prompts)
hostbox login --token <token>     # Login with API token (for CI/CD)
hostbox logout                    # Clear credentials
hostbox whoami                    # Show current user + server URL

# Projects
hostbox projects                  # List all projects
hostbox projects create           # Interactive project creation
hostbox link                      # Link current directory to a project (writes .hostbox.json)
hostbox open                      # Open project dashboard in browser

# Deployments
hostbox deploy                    # Deploy current branch (uses linked project)
hostbox deploy --branch <branch>  # Deploy specific branch
hostbox deploy --prod             # Force production deployment
hostbox status                    # Show latest deployment status
hostbox logs                      # Stream latest deployment logs
hostbox logs <deployment-id>      # Stream specific deployment logs
hostbox rollback <deployment-id>  # Rollback to specific deployment
hostbox rollback --last           # Rollback to previous production deployment

# Domains
hostbox domains                   # List domains for linked project
hostbox domains add <domain>      # Add custom domain
hostbox domains remove <domain>   # Remove custom domain
hostbox domains verify <domain>   # Verify domain DNS

# Environment Variables
hostbox env                       # List env vars (secrets masked)
hostbox env set <KEY=value>       # Set env var
hostbox env set --secret <KEY=value>  # Set secret env var
hostbox env delete <KEY>          # Delete env var
hostbox env import <file.env>     # Bulk import from .env file
hostbox env export                # Export non-secret vars as .env

# Admin (admin users only)
hostbox admin reset-password <email>   # Reset user password
hostbox admin backup                   # Create database + config backup
hostbox admin backup --output <path>   # Backup to specific path
```

### 7.3 Project Linking

Running `hostbox link` in a project directory creates `.hostbox.json`:
```json
{
  "project_id": "abc123",
  "project_slug": "my-app"
}
```

This allows subsequent commands (`deploy`, `env`, `domains`, etc.) to run without specifying `--project`.

### 7.4 Config File

```yaml
# ~/.config/hostbox/config.yaml
server: https://hostbox.example.com
token: <stored securely via OS keyring, fallback to file>
```

On headless servers (no keyring available), the token is stored in the config file with `0600` permissions.

---

## 8. GitHub Integration

### 8.1 GitHub App

**Setup**: The setup wizard guides users through creating a GitHub App with a pre-filled manifest URL, minimizing manual configuration.

**Permissions**
- Contents: Read (clone repos)
- Pull requests: Read & Write (post preview URLs as comments)
- Commit statuses: Read & Write (update deployment status checks)
- Deployments: Read & Write (GitHub Deployments API integration)

**Events Subscribed**
- `push` — Trigger production/branch deployments
- `pull_request` — opened, synchronize, closed, reopened
- `installation` — Track app installation/uninstallation

### 8.2 Webhook Handling

**Security**: Every webhook is verified using HMAC-SHA256 with `GITHUB_WEBHOOK_SECRET`. Invalid signatures are rejected with 401.

**Push to Production Branch**
```
Event: push
Ref: refs/heads/{production_branch}
→ Create production deployment
→ Post GitHub Deployment Status: pending → success/failure
```

**Push to Non-Production Branch (with open PR)**
```
Event: push
Ref: refs/heads/{feature-branch}
→ Only deploy if there's an open PR for this branch (avoids deploying random pushes)
→ Create preview deployment
→ Post GitHub Deployment Status with environment_url
```

**PR Opened / Synchronized**
```
Event: pull_request
Action: opened | synchronize
→ Create preview deployment
→ Post/update PR comment with preview URL
→ Post GitHub Deployment Status
```

**PR Closed**
```
Event: pull_request
Action: closed
→ Mark preview deployments for this branch as inactive
→ Update Caddy to stop routing (returns 410 Gone)
→ Artifacts retained per retention policy for potential rollback
```

### 8.3 GitHub Deployment Status Integration

Hostbox updates GitHub Deployment Status for each deployment:
```
pending    → Build queued
in_progress → Build started
success    → Deployment live (with environment_url)
failure    → Build failed (with log_url)
```

This shows deployment status directly in GitHub PR checks.

### 8.4 PR Comments

On preview deployment completion, Hostbox posts/updates a comment on the PR:
```markdown
🚀 **Preview Deployment Ready**

| Name | Status | URL |
|------|--------|-----|
| my-app | ✅ Ready | [Preview](https://my-app-a1b2c3d4.hostbox.example.com) |

**Commit**: abc1234 — "feat: add login page"
**Built in**: 45s
```

The comment is updated (not duplicated) on subsequent pushes.

---

## 9. Build System

### 9.1 Build Worker

The build worker runs as goroutines within the main Hostbox binary, using a bounded worker pool.

**Concurrency**: 1 concurrent build by default (configurable). On a 512MB VPS, only 1 build should run at a time. On larger VPS, this can be increased.

**Build Queue**: FIFO queue stored in SQLite. If a new push comes for the same project+branch while a build is queued/building, the old build is cancelled and replaced.

### 9.2 Framework Detection

Hostbox auto-detects the framework by analyzing `package.json` in the project root (or `root_directory`):

| Framework | Detection | Install Command | Build Command | Output Directory |
|-----------|-----------|-----------------|---------------|-----------------|
| Next.js (static) | `next` in dependencies | `npm install` | `npm run build` | `out` |
| Vite | `vite` in devDependencies | `npm install` | `npm run build` | `dist` |
| Create React App | `react-scripts` in deps | `npm install` | `npm run build` | `build` |
| Astro | `astro` in dependencies | `npm install` | `npm run build` | `dist` |
| Gatsby | `gatsby` in dependencies | `npm install` | `npm run build` | `public` |
| Nuxt (static) | `nuxt` in dependencies | `npm install` | `npm run generate` | `.output/public` |
| SvelteKit (static) | `@sveltejs/kit` in deps | `npm install` | `npm run build` | `build` |
| Hugo | `hugo` binary detected | n/a | `hugo --minify` | `public` |
| Plain HTML | `index.html` in root | n/a | n/a | `.` |
| Unknown | Fallback | `npm install` | `npm run build` | `dist` |

**Package Manager Detection**:
- `pnpm-lock.yaml` → `pnpm install`
- `yarn.lock` → `yarn install --frozen-lockfile`
- `bun.lockb` → `bun install`
- `package-lock.json` or fallback → `npm ci`

### 9.3 Build Container

**Base Image**: `node:{version}-slim` (configurable per project: 18, 20, 22)

**Resource Limits**
- Memory: 512MB (configurable, max 2GB)
- CPU: 1 core (configurable, max 2 cores)
- Disk: 5GB (temporary build space)
- Timeout: 10 minutes (configurable, max 30 min)
- PID limit: 256 processes

**Build Process**
```
1. Shallow clone repo (git clone --depth=1 --branch={branch})
   └── Uses GitHub App installation token for auth
2. If root_directory != "/", cd into it
3. Create Docker container from node:{version}-slim
4. Mount build cache volume: /cache/{project_id}/node_modules → /app/node_modules
5. Mount build cache volume: /cache/{project_id}/build-cache → /app/.next/cache (etc.)
6. Copy source into container
7. Set working directory
8. Inject environment variables (scoped by preview/production + built-ins)
9. Run install command (with cache)
10. Run build command
11. Copy output directory to /app/deployments/{project_id}/{deployment_id}/
12. Record artifact size
13. Destroy container
14. Clean up cloned source
```

### 9.4 Build Caching

**Node modules cache**: A named Docker volume per project stores `node_modules`. Persists across builds, dramatically reducing install times (~45s → ~5s for cache hits).

**Framework build cache**: Framework-specific caches are persisted:
- Next.js: `.next/cache`
- Vite: `node_modules/.vite`
- Gatsby: `.cache`, `public`

**Cache invalidation**: Cache volumes are reset when:
- User manually triggers "Clear cache" from dashboard
- Node version changes
- Package manager changes
- Lock file hash changes (detected by comparing SHA of lock file)

### 9.5 Garbage Collection

A background goroutine runs periodic cleanup:

**Artifact Cleanup** (runs every 6 hours):
- Keep last N deployments per project (configurable, default: 20)
- Keep artifacts for at most M days (configurable, default: 30)
- Never delete the currently active production deployment
- Never delete currently active preview deployments

**Log Cleanup**:
- Log files are cleaned up when their associated deployment artifacts are cleaned

**Build Cache Cleanup**:
- Cache volumes for deleted projects are removed
- Cache volumes not accessed for 30 days are removed

**Docker Cleanup**:
- Prune stopped build containers (should already be removed, but safety net)
- Prune unused build images monthly

### 9.6 Build Logs

Logs are captured in real-time from the Docker container's stdout/stderr:

```
[2024-01-15 10:30:01] ▶ Cloning repository...
[2024-01-15 10:30:03] ▶ Detected framework: Next.js
[2024-01-15 10:30:03] ▶ Node.js version: 20
[2024-01-15 10:30:03] ▶ Package manager: pnpm
[2024-01-15 10:30:03] ▶ Running: pnpm install
[2024-01-15 10:30:15] added 1234 packages in 12s
[2024-01-15 10:30:15] ▶ Running: pnpm run build
[2024-01-15 10:30:45] ▶ Build complete (30.2s)
[2024-01-15 10:30:45] ▶ Output: 4.2 MB (23 files)
[2024-01-15 10:30:46] ▶ Deploying to: https://my-app-a1b2c3d4.hostbox.example.com
[2024-01-15 10:30:46] ✅ Deployment ready!
```

Logs are simultaneously:
1. Written to disk: `/app/logs/{deployment_id}.log`
2. Streamed to connected SSE clients
3. Sent as GitHub Deployment Status updates (summary only)

### 9.7 Static File Configuration

After build, Hostbox creates a serving configuration for the deployment:

**SPA Mode** (default for React, Vue, Svelte, etc.):
- `try_files {path} /index.html`
- All routes fallback to index.html for client-side routing

**Static Mode** (for Hugo, Gatsby, Astro, plain HTML):
- `try_files {path} {path}/ =404`
- Direct file serving, 404 for missing paths

**Custom Headers** (configured per project):
- Cache-Control for static assets (`/_next/static/*`, `/assets/*`): `public, max-age=31536000, immutable`
- Cache-Control for HTML: `public, max-age=0, must-revalidate`
- Security headers: `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`

---

## 10. Caddy Configuration

### 10.1 Overview

Caddy handles all HTTP/HTTPS traffic with automatic certificate management. Key advantages:
- Automatic SSL certificate provisioning and renewal
- Hot-reload via admin API (no restart needed)
- Efficient static file serving
- Minimal memory footprint (~20MB)

### 10.2 Communication: Hostbox ↔ Caddy

Hostbox manages Caddy's configuration via the **Caddy Admin API** (localhost:2019), not by writing Caddyfile snippets. This enables:
- Zero-downtime routing updates
- Atomic configuration changes
- No file-watching complexity

**Flow**:
1. Deployment completes → Hostbox builds Caddy JSON config for the route
2. Hostbox POSTs to `http://localhost:2019/config/apps/http/servers/...`
3. Caddy applies the new route immediately
4. On Hostbox startup, full Caddy config is regenerated from database state

### 10.3 Base Caddyfile (bootstrap only)

The initial Caddyfile is minimal — just enough to start Caddy and enable the admin API:

```caddy
{
    admin localhost:2019
    email {$ACME_EMAIL:admin@example.com}
}

# Platform domain — reverse proxy to Hostbox API + embedded dashboard
{$PLATFORM_DOMAIN} {
    # API routes
    handle /api/* {
        reverse_proxy localhost:8080
    }

    # Dashboard (served by Go binary)
    handle {
        reverse_proxy localhost:8080
    }
}
```

All deployment routes and custom domain routes are managed dynamically via the Admin API.

### 10.4 Dynamic Route Configuration (via Admin API)

**Preview deployment route** (added when deployment completes):
```json
{
  "match": [{"host": ["my-app-a1b2c3d4.hostbox.example.com"]}],
  "handle": [
    {"handler": "encode", "encodings": {"gzip": {}}},
    {"handler": "headers", "response": {"set": {
      "Cache-Control": ["public, max-age=0, must-revalidate"],
      "X-Frame-Options": ["DENY"],
      "X-Content-Type-Options": ["nosniff"]
    }}},
    {"handler": "rewrite", "uri": "{http.request.uri}"},
    {"handler": "file_server", "root": "/app/deployments/{project_id}/{deployment_id}"}
  ],
  "terminal": true
}
```

**Custom domain route**:
```json
{
  "match": [{"host": ["myapp.com", "www.myapp.com"]}],
  "handle": [
    {"handler": "encode", "encodings": {"gzip": {}}},
    {"handler": "file_server", "root": "/app/deployments/{project_id}/{active_deployment_id}"}
  ],
  "terminal": true
}
```

### 10.5 SSL/ACME Strategy

| Domain Type | Certificate Strategy | Challenge |
|-------------|---------------------|-----------|
| Platform domain | Single cert for `{PLATFORM_DOMAIN}` | HTTP-01 |
| Preview URLs | Wildcard cert `*.{PLATFORM_DOMAIN}` | DNS-01 (requires DNS provider API) |
| Custom domains | Individual certs per domain | HTTP-01 (automatic via Caddy) |

**DNS-01 for wildcards**: Requires a DNS provider module in Caddy (e.g., Cloudflare, Route53, DigitalOcean DNS). The install script builds Caddy with the appropriate DNS module based on user's DNS provider.

**Fallback**: If DNS-01 is not configured, preview URLs use individual certs per subdomain via HTTP-01 (works but slower for many previews).

### 10.6 Static Asset Caching

Caddy is configured to set appropriate cache headers:
- `/_next/static/*`, `/static/*`, `/assets/*`: `Cache-Control: public, max-age=31536000, immutable`
- `*.html`, `/`: `Cache-Control: public, max-age=0, must-revalidate`
- `*.js`, `*.css` (hashed filenames): `Cache-Control: public, max-age=31536000, immutable`

---

## 11. Security

### 11.1 Threat Model

| Threat | Mitigation |
|--------|------------|
| Unauthorized API access | JWT + refresh tokens, rate limiting |
| Brute-force login | Rate limiting (10/min per IP), account lockout after 10 failures |
| SQL injection | Parameterized queries (Go `database/sql`) — no raw string concatenation |
| XSS | React's built-in escaping, CSP headers, no `dangerouslySetInnerHTML` |
| CSRF | SameSite=Strict cookies, Origin header validation |
| Build container escape | Unprivileged containers, seccomp, no Docker socket mount |
| Secret exfiltration | AES-256-GCM encryption at rest, secrets never returned via API |
| Domain hijacking | DNS resolution verification before SSL provisioning |
| Webhook spoofing | HMAC-SHA256 signature verification on every GitHub webhook |
| Token theft | Short-lived access tokens (15min), refresh token rotation |
| Stale sessions | Refresh tokens tracked in DB, expired sessions auto-cleaned |

### 11.2 Build Container Security

```
Security constraints applied to every build container:
├── --security-opt=no-new-privileges     # Cannot escalate privileges
├── --cap-drop=ALL                       # Drop all Linux capabilities
├── --read-only (root filesystem)        # Root FS is read-only
├── --tmpfs /tmp:size=512m               # Writable /tmp with size limit
├── --network=bridge (limited)           # Network access for package downloads
├── --pids-limit=256                     # Prevent fork bombs
├── --memory=512m                        # Memory limit (configurable)
├── --cpus=1                             # CPU limit (configurable)
├── --ulimit nofile=1024:1024            # File descriptor limit
└── No Docker socket access              # Cannot interact with host Docker
```

**Network Policy**: Build containers can reach the internet (needed for `npm install`, downloading binaries). This is a pragmatic trade-off — restricting to specific registries is a post-v1 enhancement.

### 11.3 Secrets Management

- Encryption: AES-256-GCM with a master key stored in the `.env` file (or environment variable)
- Each secret gets a unique nonce
- Master key rotation: when rotated, all secrets are re-encrypted (admin CLI command)
- Secrets are decrypted only at build time, injected into the container, and never logged
- Build logs are scrubbed for known secret patterns (credit card regex, API key formats)

### 11.4 HTTP Security Headers

Applied to all responses:
```
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=()
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
```

### 11.5 Data Safety

- Database file (`hostbox.db`) permissions: `0600` (owner read/write only)
- Config file (`.env`) permissions: `0600`
- Deployment artifacts: `0644` (world-readable, since they're public static files)
- Build logs: `0640` (owner + group read)

---

## 12. Project Structure

```
hostbox/
├── cmd/
│   ├── api/                    # Main server entrypoint (API + Worker + embedded dashboard)
│   │   └── main.go
│   └── cli/                    # CLI entrypoint (separate binary)
│       └── main.go
│
├── internal/
│   ├── api/
│   │   ├── handlers/           # HTTP handlers (auth, projects, deployments, etc.)
│   │   ├── middleware/         # Auth, rate-limit, logging, CORS, recovery
│   │   ├── routes/             # Route registration
│   │   └── server.go           # Server setup, graceful shutdown
│   │
│   ├── models/                 # Database models (Go structs)
│   ├── dto/                    # Request/Response DTOs (validation tags)
│   │
│   ├── services/
│   │   ├── auth/               # Authentication (JWT, bcrypt, sessions)
│   │   ├── project/            # Project CRUD, framework detection
│   │   ├── deployment/         # Deployment orchestration, rollback
│   │   ├── build/              # Build execution, log streaming
│   │   ├── domain/             # Domain verification, Caddy config
│   │   ├── github/             # GitHub App, webhook handling, PR comments
│   │   ├── envvar/             # Env var encryption/decryption
│   │   ├── notification/       # Discord, Slack, webhook notifications
│   │   └── cleanup/            # Garbage collection (artifacts, logs, caches)
│   │
│   ├── repository/             # Database queries (SQLite via database/sql)
│   │
│   ├── worker/                 # Build worker pool (goroutine-based)
│   │   ├── pool.go             # Bounded worker pool
│   │   ├── queue.go            # Job queue (SQLite-backed)
│   │   └── executor.go         # Build execution logic
│   │
│   ├── platform/
│   │   ├── caddy/              # Caddy Admin API client
│   │   ├── docker/             # Docker SDK wrapper (build containers)
│   │   └── detect/             # Framework/package-manager detection
│   │
│   └── config/                 # Configuration loading (.env, defaults)
│
├── migrations/                 # SQL migrations (embedded in binary)
│   ├── 001_initial.sql
│   └── ...
│
├── web/                        # Vite + React dashboard
│   ├── src/
│   │   ├── app/                # Page components
│   │   ├── components/         # UI components
│   │   ├── lib/                # API client, hooks, utils
│   │   └── main.tsx
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── docker/
│   ├── Dockerfile              # Multi-stage: builds Go binary + web dashboard
│   ├── caddy/
│   │   ├── Caddyfile           # Bootstrap Caddyfile
│   │   └── Dockerfile          # Custom Caddy with DNS modules
│   └── docker-compose.yml      # Production deployment
│
├── scripts/
│   ├── install.sh              # One-line installer for fresh VPS
│   ├── update.sh               # Self-update script
│   └── build.sh                # Local development build script
│
├── SPEC.md
├── README.md
├── .env.example
├── go.mod
├── go.sum
└── Makefile
```

---

## 13. Docker Compose (Production)

```yaml
services:
  hostbox:
    build:
      context: .
      dockerfile: docker/Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DATABASE_PATH=/app/data/hostbox.db
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - JWT_SECRET=${JWT_SECRET}
      - GITHUB_APP_ID=${GITHUB_APP_ID}
      - GITHUB_APP_PEM=${GITHUB_APP_PEM}
      - GITHUB_WEBHOOK_SECRET=${GITHUB_WEBHOOK_SECRET}
      - PLATFORM_DOMAIN=${PLATFORM_DOMAIN}
    volumes:
      - hostbox-data:/app/data          # SQLite DB + config
      - deployments:/app/deployments    # Build artifacts
      - logs:/app/logs                  # Build logs
      - build-cache:/cache              # Node modules cache
      - /var/run/docker.sock:/var/run/docker.sock  # For build containers
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 5s
      retries: 3

  caddy:
    build:
      context: ./docker/caddy
      dockerfile: Dockerfile
    ports:
      - "80:80"
      - "443:443"
    environment:
      - PLATFORM_DOMAIN=${PLATFORM_DOMAIN}
      - ACME_EMAIL=${ACME_EMAIL}
    volumes:
      - ./docker/caddy/Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
      - caddy-config:/config
      - deployments:/app/deployments:ro  # Read-only access to artifacts
    depends_on:
      hostbox:
        condition: service_healthy
    restart: unless-stopped

volumes:
  hostbox-data:
  deployments:
  logs:
  build-cache:
  caddy-data:
  caddy-config:
```

**Note**: No PostgreSQL, no Redis, no separate web container, no separate worker. Just 2 services.

### 13.1 Docker Compose (Development)

```yaml
services:
  hostbox:
    build:
      context: .
      dockerfile: docker/Dockerfile
      target: dev   # Uses air for hot-reload
    ports:
      - "8080:8080"
    volumes:
      - .:/app
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - DATABASE_PATH=/app/data/hostbox.db
      - LOG_LEVEL=debug

  web:
    # During development, run Vite dev server separately for HMR
    build:
      context: ./web
      dockerfile: Dockerfile.dev
    ports:
      - "3000:3000"
    volumes:
      - ./web:/app
      - /app/node_modules
    environment:
      - VITE_API_URL=http://localhost:8080/api/v1

  caddy:
    image: caddy:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./docker/caddy/Caddyfile.dev:/etc/caddy/Caddyfile:ro
```

---

## 14. Environment Variables

### Platform Configuration (.env)

```bash
# ─── Required ───────────────────────────────────────────────

# Platform domain (where Hostbox is accessible)
PLATFORM_DOMAIN=hostbox.example.com

# Encryption key for secrets (generate: openssl rand -hex 32)
ENCRYPTION_KEY=your-64-char-hex-key

# JWT signing secret (generate: openssl rand -hex 32)
JWT_SECRET=your-64-char-hex-key

# ─── GitHub App ─────────────────────────────────────────────

# GitHub App credentials (create at https://github.com/settings/apps)
GITHUB_APP_ID=123456
GITHUB_APP_SLUG=hostbox
GITHUB_APP_PEM="-----BEGIN RSA PRIVATE KEY-----\n..."
GITHUB_WEBHOOK_SECRET=your-webhook-secret

# ─── Optional ───────────────────────────────────────────────

# Database (default: /app/data/hostbox.db)
DATABASE_PATH=/app/data/hostbox.db

# ACME email for Let's Encrypt certificate notifications
ACME_EMAIL=admin@example.com

# DNS provider for wildcard certs (optional, for preview URL wildcards)
# Supported: cloudflare, digitalocean, route53, vultr
DNS_PROVIDER=cloudflare
DNS_API_TOKEN=your-dns-api-token

# Email (optional — if not configured, email verification is disabled)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=hostbox@example.com
SMTP_PASS=password
EMAIL_FROM=noreply@example.com

# Logging
LOG_LEVEL=info  # debug, info, warn, error

# Build defaults (can be overridden per project)
MAX_CONCURRENT_BUILDS=1
DEFAULT_BUILD_TIMEOUT_MINUTES=10
DEFAULT_BUILD_MEMORY_MB=512
```

---

## 15. Installation & Deployment

### 15.1 One-Line Install

```bash
curl -sSL https://get.hostbox.sh | bash
```

The install script:
1. Checks system requirements (OS, architecture, Docker)
2. Installs Docker if not present
3. Downloads latest Hostbox release (docker-compose.yml + .env.example + Caddyfile)
4. Prompts for: platform domain, ACME email, DNS provider (optional)
5. Generates secrets (ENCRYPTION_KEY, JWT_SECRET)
6. Starts services via `docker compose up -d`
7. Prints: "Visit https://{domain} to complete setup"

### 15.2 System Requirements

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| RAM | 512MB | 1GB |
| Disk | 10GB | 20GB+ |
| CPU | 1 vCPU | 2 vCPU |
| OS | Ubuntu 20.04+ / Debian 11+ | Ubuntu 22.04 LTS |
| Docker | 20.10+ | Latest stable |
| Network | Public IPv4, ports 80/443 open | + IPv6 |

### 15.3 Self-Update

```bash
hostbox admin update        # CLI command
# or
curl -sSL https://get.hostbox.sh/update | bash
```

The update process:
1. Pulls latest Docker images
2. Runs database migrations (if any)
3. Restarts services with zero downtime (rolling restart)
4. Verifies health check passes

### 15.4 Backup & Restore

**Backup** (creates tarball):
```bash
hostbox admin backup --output /backups/hostbox-$(date +%Y%m%d).tar.gz
```

Includes:
- SQLite database
- .env configuration
- Caddy certificates (from caddy-data volume)
- Does NOT include deployment artifacts (can be rebuilt from Git)

**Restore**:
```bash
hostbox admin restore --input /backups/hostbox-20240115.tar.gz
```

**Automated backups**: Configure via cron or optional Litestream integration for continuous SQLite replication to S3.

### 15.5 Deployment Checklist

1. ☐ VPS with public IP, Docker installed, ports 80/443 open
2. ☐ Domain DNS A record pointing to VPS IP
3. ☐ Wildcard DNS `*.{domain}` pointing to VPS IP (for preview URLs)
4. ☐ Run install script
5. ☐ Complete setup wizard in browser
6. ☐ Create GitHub App (guided by setup wizard)
7. ☐ Configure GitHub App credentials in settings
8. ☐ Install GitHub App on target repositories
9. ☐ Create first project and push to trigger deployment
10. ☐ (Optional) Configure DNS provider for wildcard SSL
11. ☐ (Optional) Configure SMTP for email notifications
12. ☐ (Optional) Set up automated backups

---

## 16. Comparison with Alternatives

| Feature | Hostbox | Coolify | Dokploy | Vercel |
|---------|---------|---------|---------|--------|
| Min RAM | **512MB** | 2GB | 2GB | N/A (cloud) |
| Min Disk | **10GB** | 30GB | 40GB | N/A (cloud) |
| Container count | **2** | 5+ | 4+ | N/A |
| Database | **SQLite** | PostgreSQL | PostgreSQL | N/A |
| Focus | Frontend/static | Full-stack | Full-stack | Frontend |
| Docker Compose deploy | ✗ | ✓ | ✓ | ✗ |
| Database hosting | ✗ | ✓ | ✓ | ✓ (add-on) |
| Multi-server | ✗ | ✓ | ✓ | ✓ |
| Preview deploys | ✓ | ✓ | ✓ | ✓ |
| Custom domains | ✓ | ✓ | ✓ | ✓ |
| Auto SSL | ✓ | ✓ | ✓ | ✓ |
| Build caching | ✓ | ✓ | ✓ | ✓ |
| Framework detection | ✓ | ✗ | ✗ | ✓ |
| One-line install | ✓ | ✓ | ✓ | N/A |
| Self-hostable | ✓ | ✓ | ✓ | ✗ |
| Price | Free | Free | Free | Freemium |

---

## 17. Future Considerations (Post-v1)

### v1.1 (Near-term)
- Team/organization support with RBAC
- GitLab / Bitbucket support
- Deploy from CLI without GitHub (upload tarball)
- Custom build Docker images
- Monorepo support with automatic detection
- Deploy protection rules (require approval for production)
- Basic bandwidth/request analytics per project

### v1.2 (Medium-term)
- Multiple build workers (fan-out builds across cores)
- Edge functions / serverless functions (Deno / Bun runtime)
- A/B deployment (traffic splitting between two versions)
- Branch-specific environment variables
- Webhooks for deployment lifecycle (for external CI/CD integration)
- Litestream integration for continuous DB backup
- Telegram / email notification channels

### v2.0 (Long-term)
- Multi-server support (deploy to remote VPS)
- Full-stack support (Node.js server, Go, Python backends)
- Managed database add-ons (PostgreSQL, Redis)
- Template marketplace
- Plugin system for custom build providers
- Usage metering and billing (for hosting providers)

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| Preview Deployment | Temporary URL for testing PR/branch changes |
| Production Deployment | Live URL served on the production branch and custom domains |
| Artifact | Build output files (HTML, JS, CSS, images) — static files only |
| Rollback | Reverting to a previous deployment's artifacts (instant, no rebuild) |
| GitHub App | An application installed on GitHub repos that receives webhooks and can access repo contents |
| Framework Detection | Automatic identification of build tool (Next.js, Vite, etc.) from package.json |
| Build Cache | Persisted node_modules and framework-specific caches across builds |
| Garbage Collection | Automatic cleanup of old deployment artifacts, logs, and caches |
| SPA Mode | Single-page application serving where all routes fallback to index.html |
| Static Mode | Direct file serving where missing files return 404 |
| WAL Mode | SQLite Write-Ahead Logging — enables concurrent reads during writes |

## Appendix B: Framework Detection Logic

```go
// Pseudocode for framework detection
func detectFramework(packageJSON PackageJSON) Framework {
    deps := merge(packageJSON.Dependencies, packageJSON.DevDependencies)

    switch {
    case deps["next"]:
        return Framework{Name: "nextjs", Build: "npm run build", Output: "out"}
    case deps["vite"]:
        return Framework{Name: "vite", Build: "npm run build", Output: "dist"}
    case deps["react-scripts"]:
        return Framework{Name: "cra", Build: "npm run build", Output: "build"}
    case deps["astro"]:
        return Framework{Name: "astro", Build: "npm run build", Output: "dist"}
    case deps["gatsby"]:
        return Framework{Name: "gatsby", Build: "npm run build", Output: "public"}
    case deps["nuxt"]:
        return Framework{Name: "nuxt", Build: "npm run generate", Output: ".output/public"}
    case deps["@sveltejs/kit"]:
        return Framework{Name: "sveltekit", Build: "npm run build", Output: "build"}
    default:
        return Framework{Name: "unknown", Build: "npm run build", Output: "dist"}
    }
}
```

## Appendix C: Install Script Outline

```bash
#!/bin/bash
set -euo pipefail

HOSTBOX_VERSION="latest"
INSTALL_DIR="/opt/hostbox"

echo "🚀 Installing Hostbox..."

# Check prerequisites
check_os()         # Linux only (Ubuntu/Debian)
check_arch()       # amd64 or arm64
check_root()       # Must run as root or with sudo
check_ports()      # 80, 443 must be available

# Install Docker if needed
if ! command -v docker &>/dev/null; then
    curl -fsSL https://get.docker.com | sh
fi

# Download Hostbox
mkdir -p "$INSTALL_DIR"
curl -fsSL "https://get.hostbox.sh/releases/${HOSTBOX_VERSION}/docker-compose.yml" \
    -o "$INSTALL_DIR/docker-compose.yml"
# ... download other files ...

# Generate secrets
ENCRYPTION_KEY=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)
WEBHOOK_SECRET=$(openssl rand -hex 20)

# Interactive configuration
read -p "Platform domain: " PLATFORM_DOMAIN
read -p "ACME email: " ACME_EMAIL

# Write .env
cat > "$INSTALL_DIR/.env" <<EOF
PLATFORM_DOMAIN=$PLATFORM_DOMAIN
ENCRYPTION_KEY=$ENCRYPTION_KEY
JWT_SECRET=$JWT_SECRET
GITHUB_WEBHOOK_SECRET=$WEBHOOK_SECRET
ACME_EMAIL=$ACME_EMAIL
EOF

# Start
cd "$INSTALL_DIR"
docker compose up -d

echo "✅ Hostbox installed! Visit https://$PLATFORM_DOMAIN to complete setup."
```
