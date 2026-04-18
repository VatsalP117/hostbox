# RFC: Move Dashboard to a Dedicated Subdomain

## Background & Motivation

Today, when a user installs Hostbox and points their domain (say `algomind.com`) at their server, the **root domain** is claimed by the Hostbox management dashboard. User projects are deployed to subdomains like `myapp.algomind.com`. This means:

```
algomind.com              →  Hostbox dashboard (wasted on a control panel)
myapp.algomind.com        →  User's deployed project
another.algomind.com      →  User's deployed project
```

This is a problem. The root domain is the most valuable piece of real estate — it's what people type in their browser, what search engines index, and what users associate with the product. Bundling the admin panel onto it means the user can never deploy their own project to the root, which defeats the purpose of a self-hosted deployment platform. It would be like Vercel occupying `algomind.com` for its dashboard instead of letting you put your actual site there.

### What We Want

Move the dashboard to a dedicated subdomain (defaulting to `dash.`) so the root domain is freed up:

```
dash.algomind.com          →  Hostbox dashboard (control panel + API)
algomind.com               →  Not used by Hostbox unless a project explicitly claims it
*.algomind.com             →  All project deployments as before
```

Now `algomind.com` can be used like any other custom domain — a user can claim it for their project.

### Example Walkthrough

**Before (current):**

```
User installs Hostbox → sets domain to algomind.com
DNS:  A  algomind.com       → 1.2.3.4
      A  *.algomind.com     → 1.2.3.4

algomind.com              →  "Welcome to Hostbox" (dashboard login)
myapp.algomind.com        →  User's Next.js app
```

The user can never put their own site at `algomind.com`. It's permanently occupied.

**After (proposed):**

```
User installs Hostbox → sets domain to algomind.com
DNS:  A  algomind.com       → 1.2.3.4   (root — available for projects)
      A  *.algomind.com     → 1.2.3.4   (covers dash.algomind.com + project subdomains)

dash.algomind.com          →  "Welcome to Hostbox" (dashboard login)
algomind.com               →  Unassigned (until claimed by a project)
myapp.algomind.com         →  User's Next.js app
```

If the user later adds `algomind.com` as a custom domain for a project:

```
algomind.com               →  User's production website
dash.algomind.com          →  Hostbox dashboard (still accessible for management)
myapp.algomind.com         →  User's Next.js app
```

This is exactly how platforms like Vercel work — `vercel.com` is the platform itself, while users deploy to their own root domains. With Hostbox being self-hosted, the platform and the user's domain are the same, so the platform should take a subdomain and leave the root free.

### Why a Default of `dash.`

- **Short and memorable** — easy to type, obvious purpose
- **Customizable** — users can override via `DASHBOARD_DOMAIN` env var if they prefer `admin.`, `manage.`, `console.`, etc.
- **No extra DNS entry needed** — `dash.algomind.com` is already covered by the wildcard `*.algomind.com` A record
- **Backward compatible** — existing installs that skip the new env var get `dash.{PLATFORM_DOMAIN}` automatically

---

## Effort Estimate

| Area | Files | Effort |
|------|-------|--------|
| Config | 2 | Small |
| Caddy builder | 1 | Medium |
| CORS middleware | 1 | Small |
| URL generation (GitHub + notifications) | 2 | Small |
| Install script & env templates | 3 | Small |
| Docker compose | 1 | Small |
| Test updates | ~3 | Medium |
| Marketing/docs | 4 | Small |
| **Total** | **~18** | **~1-2 days** |

---

## Implementation Plan

### Phase 1 — Config & Env

#### `internal/config/config.go`

Add `DashboardDomain` field with auto-default:

```go
DashboardDomain string `env:"DASHBOARD_DOMAIN" envDefault:""`
```

In validation, default it to `dash.{PlatformDomain}` if empty:

```go
if cfg.DashboardDomain == "" {
    cfg.DashboardDomain = "dash." + cfg.PlatformDomain
}
```

Add a `DashboardBaseURL()` method:

```go
func (c *Config) DashboardBaseURL() string {
    scheme := "https"
    if !c.PlatformHTTPS {
        scheme = "http"
    }
    return scheme + "://" + c.DashboardDomain
}
```

#### `internal/config/config_test.go`

Add tests for:
- `DashboardDomain` defaults to `dash.` + `PlatformDomain` when empty
- `DashboardDomain` uses explicit value when provided
- `DashboardBaseURL()` returns correct scheme + domain

#### `.env.example` / `.env.production.example`

Add:

```
# Dashboard domain (defaults to dash.{PLATFORM_DOMAIN})
DASHBOARD_DOMAIN=
```

#### `scripts/install.sh`

Update `configure()` to prompt for the dashboard host with a default:

```bash
read -rp "Dashboard host [dash.${DOMAIN}]: " DASHBOARD_INPUT
DASHBOARD_DOMAIN="${DASHBOARD_INPUT:-dash.${DOMAIN}}"
```

Write `DASHBOARD_DOMAIN=${DASHBOARD_DOMAIN}` to the `.env` file.

Update `print_success()` to show:

```
Dashboard:  https://${DASHBOARD_DOMAIN}
```

Update the DNS instructions to mention the dashboard subdomain is covered by the wildcard.

#### `cmd/api/main.go`

Wire `DashboardDomain` into the `ConfigBuilder` alongside `PlatformDomain`.

---

### Phase 2 — Caddy Routing

#### `internal/services/caddy/builder.go`

This is the core change. Two modifications:

**1. Change platform route to match `DashboardDomain` instead of `PlatformDomain`:**

Currently `buildPlatformRoute()` matches `cfg.PlatformDomain` and reverse-proxies to the API server. Change the host matcher to `cfg.DashboardDomain`:

```go
func (cb *ConfigBuilder) buildPlatformRoute() Route {
    // Host match: cfg.DashboardDomain instead of cfg.PlatformDomain
    // Reverse proxy to APIUpstream stays the same
}
```

**2. Update TLS policies:**

Route ordering in `BuildFullConfig()` becomes:

1. Dashboard route (`dash.algomind.com` → API)
2. Custom domain routes
3. Project deployment routes (production, preview, branch)
4. TLS automation policies

If using DNS-01 challenge (wildcard cert), `dash.algomind.com` is already covered by `*.algomind.com` — no change needed.

If using HTTP-01 challenge (no DNS provider), use `DashboardDomain` as the primary platform subject for Hostbox routes.

---

### Phase 3 — API Server & Middleware

#### `internal/api/server.go`

Pass explicit allowed origins to the CORS middleware:

- `cfg.DashboardBaseURL()` for the dashboard SPA origin

#### `internal/api/middleware/cors.go`

Restrict API browser-origin access to the dashboard origin:

```go
allowedOrigins := []string{cfg.DashboardBaseURL()}
```

---

### Phase 4 — URL Generation

Most URL generation does **not** change — deployment URLs (`myapp.algomind.com`) use `PlatformDomain`, which is correct. Only dashboard-facing links need updating:

| File | Change |
|------|--------|
| `internal/services/github/comments.go` | PR comment footer links to dashboard → use `DashboardDomain` |
| `cmd/api/main.go` | Notification `serverURL` should use `cfg.DashboardBaseURL()` instead of `https://{PlatformDomain}` |

No dashboard-domain change needed in `internal/api/handlers/domains.go` because DNS instructions for custom domains should still point at the platform ingress domain.

No changes needed in:
- `internal/worker/util.go` — deployment URLs use subdomains of `PlatformDomain`
- `internal/services/deployment/service.go` — rollback/promote URLs are deployment URLs
- `internal/services/github/status.go` — GitHub deployment status targets are deployment URLs

---

### Phase 5 — Docker

#### `docker-compose.yml`

No extra env mapping is required if `.env` already includes `DASHBOARD_DOMAIN` and `hostbox` uses `env_file`.

#### `docker-compose.dev.yml`

Set `DASHBOARD_DOMAIN=dash.localhost` for `hostbox-dev`.

---

### Phase 6 — Tests

#### `internal/services/caddy/builder_test.go`

- Add `DashboardDomain` (`dash.example.com`) to all test fixtures
- Assert the platform route host matcher is `dash.example.com` (not `example.com`)
- Verify route ordering: dashboard route first, then custom domains, then project deployments

#### `internal/config/config_test.go`

- Test `DashboardDomain` default behavior
- Test `DashboardBaseURL()` method

#### `internal/api/middleware/cors_test.go`

- Add coverage for dashboard-origin allowlist behavior

---

### Phase 7 — Docs & Marketing

| File | Change |
|------|--------|
| `README.md` | Update "Access Dashboard" to reference `dash.your-domain.com` |
| `docs/SELF-HOSTING.md` | Add DNS setup note for dashboard subdomain |
| `marketing/index.html` | Update setup instructions |
| `marketing/cli.html` | Update setup instructions |
| `marketing/features.html` | Update domain references |

---

## Complete File Change List

```
internal/config/config.go                    — DashboardDomain field + DashboardBaseURL()
internal/config/config_test.go               — Test new field
internal/services/caddy/builder.go           — Platform route uses DashboardDomain
internal/services/caddy/builder_test.go      — Update test fixtures + assertions
internal/services/github/comments.go          — Dashboard links use DashboardDomain
internal/api/server.go                        — Pass DashboardDomain to CORS
internal/api/middleware/cors.go               — Restrict to dashboard origin
internal/api/middleware/cors_test.go          — CORS dashboard-origin coverage
cmd/api/main.go                              — Wire DashboardDomain into ConfigBuilder + notification dashboard URL
scripts/install.sh                            — Prompt for dashboard host
.env.example                                  — Add DASHBOARD_DOMAIN
.env.production.example                       — Add DASHBOARD_DOMAIN
docker-compose.dev.yml                        — Pass DASHBOARD_DOMAIN
README.md                                     — Update dashboard URL
docs/SELF-HOSTING.md                          — Update DNS instructions
marketing/index.html                          — Update domain references
marketing/cli.html                            — Update domain references
marketing/features.html                       — Update domain references
```

**Total: ~18 files, estimated 1-2 days of work.**

---

## Migration Note for Existing Installs

For anyone already running Hostbox with the root domain as the dashboard:

1. Add `DASHBOARD_DOMAIN=dash.<your-domain>` to `.env`
2. Update docker-compose images
3. Run `docker compose pull && docker compose up -d`
4. Update any bookmarks to use `dash.<your-domain>` instead of the root

The root domain will no longer serve Hostbox by default, so old root-domain dashboard bookmarks must be updated to `dash.<your-domain>`.

---

## Future: Root Domain as a Project

Once the dashboard is off the root domain, a user can claim `algomind.com` as a custom domain for one of their projects — no code change needed. The existing custom domain flow already supports this:

1. Add `algomind.com` as a custom domain to a project
2. Verify DNS (A record already points to the server)
3. Caddy picks it up via the existing custom domain route handler

No route conflict handling is needed for root-domain redirects because Hostbox will not install a root redirect route.
