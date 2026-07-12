# Hostbox v0 → v1 Readiness Roadmap

**Audit date:** 2026-07-12

**Current status:** v0 / incomplete; not production-ready

**Target:** a dependable, lightweight, self-hosted Vercel-like experience for frontend and static builds on one VM

## 1. Executive conclusion

Hostbox has the correct foundation for the intended product: one Go API/worker process, SQLite, Caddy, a React dashboard, a Go CLI, and short-lived Docker build containers. This is materially lighter than a general-purpose PaaS such as Coolify or Dokploy because it does not run PostgreSQL, Redis, a permanent application-container fleet, or a multi-service control plane.

The repository is beyond a prototype, but it is not v1 yet. Unit-tested components exist for most subsystems, while the critical user journey—install on a clean VM, connect GitHub, push a frontend repository, watch a deterministic build, receive a working preview URL, promote/rollback it, and recover the instance—has not been proven end to end. Several paths are demonstrably incomplete or unsafe.

The shortest credible path to v1 is to **finish and prove the existing single-node/static architecture**, not add more platform breadth.

## 2. v1 product contract

### v1 promise

A user can install Hostbox on a clean Linux VM, complete setup in the browser, connect a GitHub App, deploy supported static/frontend projects on push or pull request, use preview and production URLs with HTTPS, attach custom domains, inspect logs, manage environment variables, and reliably promote, rollback, back up, restore, and upgrade the instance.

### Supported v1 shape

- One Hostbox instance on one Linux VM.
- One Go control-plane process plus Caddy; SQLite for metadata.
- Docker is required for builds but no user application container runs after a build.
- GitHub App integration only.
- Static HTML and static output from explicitly supported frameworks.
- Local artifacts and a single bounded build queue.
- Dashboard and CLI as two clients of the same versioned API.

### Explicit v1 non-goals

- SSR, server functions, edge functions, ISR, or long-running app containers.
- Databases, Redis, object storage, cron jobs, or background workers for user apps.
- Arbitrary Dockerfiles or Docker Compose deployments.
- GitLab, Bitbucket, Kubernetes, multi-node workers, HA, or horizontal scaling.
- Team organizations, billing, quotas by plan, or a template marketplace.
- A general-purpose replacement for Coolify/Dokploy.

The UI, README, marketing site, framework detector, and errors must all state these boundaries consistently. Unsupported SSR projects must fail before installing dependencies with an actionable static-export message.

## 3. Repository-grounded baseline

### What is already real

- Go/Echo API with authentication, ownership checks, SQLite repositories/migrations, rate limiting, and structured errors.
- In-process bounded worker pool, Docker resource constraints, build logs, SSE streaming, crash cleanup, and build cancellation primitives.
- Static framework/package-manager detection, monorepo detection, environment-variable injection, and cache volumes.
- Caddy config generation, preview/production/branch/custom-domain route types, automatic TLS configuration, startup sync, and periodic reconciliation.
- GitHub App manifest setup, installation token acquisition, repository listing, signed webhooks, push/PR handlers, and private-repository cloning.
- React dashboard covering setup, auth, projects, deployments, logs, domains, environment variables, notifications, administration, and monitoring.
- Go CLI commands for auth, projects, deployments, domains, environment variables, and backups.
- Install/update scripts, Docker packaging, and GitHub Actions test/release workflows.
- Backend tests currently pass; Go vet and normal local builds are available.

### Confirmed release blockers and drift

| Area | Current evidence | v1 consequence |
| --- | --- | --- |
| Release binaries | `.github/workflows/release.yml` builds the API with `CGO_ENABLED=0`, but `go-sqlite3` requires CGO; the resulting binary exits when opening SQLite. | Published standalone server archives are unusable. |
| CLI installer | Release archives contain a suffixed CLI binary, while `scripts/install-cli.sh` tries to move an extracted file named `hostbox`. | One-line CLI installation fails. |
| CLI deployments | The CLI posts to the record-only deployment endpoint, and its single-deployment URL does not exist in the router. | `hostbox deploy` can create a permanently queued record; status lookup is broken. |
| Deployment integrity | `internal/worker/executor.go` clones a branch head but never checks out or verifies the deployment commit SHA. | The built code may differ from the commit recorded in Hostbox/GitHub. |
| Rollback/promote | `internal/services/deployment/service.go` creates ready records but has TODOs instead of updating Caddy. | “Instant rollback” and promotion do not change live production traffic immediately. |
| Cancellation | The service writes `cancelled`, while the cancelled executor can subsequently write `failed`. | Final state and notifications are race-dependent. |
| Framework serving | Detection results are not persisted to the project, but Caddy SPA fallback depends on `project.Framework`. | Auto-detected Vite/CRA/etc. deployments can serve `/` but fail on client-side deep links. |
| GitHub feedback | Status reporter and PR comment manager exist but are not constructed or called by `cmd/api/main.go`. | GitHub does not receive the promised pending/success/failure state or preview comment. |
| Webhook reliability | Webhooks launch an unbounded goroutine using `context.Background()`; delivery IDs are not persisted/deduplicated. | Accepted events can be lost on restart and duplicate deliveries are not handled durably. |
| Branch cleanup | Branch-deletion pushes are ignored; PR close removes deployment routes but not the branch-stable route. | Stale preview routes can remain live. |
| Build path isolation | `root_directory` and output paths are joined without the existing `SafeJoinPath` guard. | A project setting can escape the cloned checkout and expose host-mounted paths to a build. |
| Outbound webhooks | `ValidateWebhookURL` exists but notification create/update handlers do not use it. | Notification delivery has an SSRF path, including DNS-rebinding considerations. |
| Dashboard/API contract | `web/src/hooks/use-user.ts` calls `/profile`, `/profile/password`, and session endpoints that are absent; the backend uses `/auth/me` and `/auth/me/password`. | Profile, password, and session UI paths fail at runtime. |
| Frontend gate | Dashboard lint currently fails on two `no-control-regex` errors; no browser tests exist. | Current CI contract is not green and UI behavior is unproved. |
| Marketing build | A clean `npm ci` in `marketing/` fails because Astro 6 conflicts with the Tailwind integration peer range; marketing is not in CI. | The public site is not reproducibly buildable. |
| Dependency security | `npm audit --omit=dev` reports four runtime findings (two high, two moderate), including `ansi-to-react`/`linkify-it` and React Router. | Known fixable runtime advisories remain in the dashboard lockfile. |
| Domain verification | Verification only checks that the name resolves, not that it targets this Hostbox instance; route changes rely on the five-minute full sync. | An unrelated DNS target can be accepted and domain changes are delayed. |
| Settings | Admin edits for maximum projects/concurrent builds are stored, but project creation does not enforce the project limit and the live worker pool uses startup env configuration. Defaults also differ between schema, API, and UI. | Settings can claim a limit that is not active. |
| Restore/update | Restore closes and replaces the live DB but does not actually restart/reopen the process. Update execution is not routed and its rollback command is invalid. | Recovery and upgrade promises are unsafe. |
| Cleanup | Project deletion cascades DB rows but does not synchronously remove routes, artifacts, logs, or cache volumes; GC only iterates existing projects. | Deleted projects can leave data and routes behind. |
| Packaging hygiene | A platform-specific 18 MB `api` executable and generated frontend build metadata are tracked. | Repository/release inputs contain stale machine-generated artifacts. |
| Resource claim | The 512 MB VM goal is not benchmarked. Default build memory is 512 MB and workspaces can request 1 GB, before Docker/OS/Hostbox/Caddy overhead. | The headline lightweight claim is currently unverified and internally inconsistent. |

## 4. Priority model

- **P0 — v1 blocker:** security boundary, data loss, incorrect deployment, broken primary journey, broken release/install, or no proof of the promise.
- **P1 — required for v1 quality:** operability, clear failure handling, supported-client completeness, and repeatable administration.
- **P2 — complete before final v1 tag when practical:** polish, performance optimization, developer experience, and documentation consistency that does not block a release candidate.

## 5. Workstreams

### A. Lock the product and API contract

**P0**

- Replace aspirational feature claims with a checked v1 support matrix: framework/version, required static-export configuration, package manager, Node version, monorepo behavior, preview support, and known limits.
- Decide whether one GitHub repository may map to one project or many. Enforce uniqueness if it is one; otherwise make webhook lookup fan out deterministically by repository and installation.
- Remove or merge the record-only `POST /projects/:id/deployments` path. There must be one command that atomically creates and enqueues a deployment.
- Define the deployment state machine, legal transitions, idempotency rules, ownership rules, and error codes. Enforce transitions in the repository/service layer rather than in callers.
- Version and document one API contract used by dashboard and CLI. Add schema/contract generation or tests so clients cannot silently drift.
- Decide the v1 account model. The existing multi-user registration/ownership surface must either be fully supported and tested or explicitly reduced to single-admin mode.

**Exit criteria**

- The support matrix, API route table, UI, CLI help, README, and marketing claims agree.
- Unsupported projects are rejected early and do not create stuck deployments.

### B. Close the build security boundary

**P0**

- Validate and canonicalize `root_directory` and `output_directory`; reject absolute paths, `..`, symlink escapes, NUL/control characters, and any path outside the cloned repository/output root. Use the existing safe-path helper in the real worker path and test symlinks.
- Check out the exact requested commit SHA after fetching the allowed ref, verify that it belongs to the intended repository/ref, and record the resolved SHA. Never report one SHA while building another.
- Remove installation tokens from command arguments and persisted Git remotes. Use a short-lived credential helper/header, redact all subprocess errors, and delete credentials before build execution.
- Threat-model the Docker socket. Keep the control plane as the only component with socket access; document that a Hostbox process compromise is host-root equivalent. Evaluate a restricted Docker socket proxy without adding unacceptable idle overhead.
- Keep build containers unprivileged: non-root user, no-new-privileges, minimal capabilities, read-only root, PID/CPU/memory/file-size/time limits, bounded writable tmp/cache, and no Docker socket.
- Define network policy. At minimum block link-local/cloud metadata and private control-plane networks; decide how package registries and user build-time APIs remain reachable.
- Define preview-secret behavior for untrusted/fork PRs. Default to no secrets for fork PRs and make any override explicit and visibly dangerous.
- Apply SSRF validation on notification URLs at create, update, test, and send time. Resolve every connection target, cover IPv4/IPv6/private/special-use ranges, redirects, and DNS rebinding.
- Normalize and validate domains, branches, repository names, build commands, environment keys, and all file paths at the boundary. Add fuzz/property tests for path and archive extraction.
- Add a Content Security Policy for the dashboard and review token-in-query SSE exposure. Prefer an authenticated fetch stream or a short-lived, deployment-scoped stream token so access JWTs do not appear in URLs/logs/history.

**P1**

- Redact secrets from build logs using known secret values plus common token patterns, while preserving useful errors.
- Add dependency, Go vulnerability, container-image, secret, and license scans to CI with a documented severity policy.
- Remove the known runtime npm advisories and establish automated dependency updates.
- Document the security model: trusted instance admins/project owners, untrusted repository code, GitHub webhook trust, local-VM trust, and non-goals.

**Exit criteria**

- Security tests prove that project paths, archives, symlinks, notification URLs, logs, and build networking cannot cross the documented boundary.
- A build cannot read arbitrary host files or instance secrets beyond variables intentionally injected into that deployment.

### C. Make deployments deterministic and atomic

**P0**

- Make create + enqueue transactional/idempotent. A queue failure must not leave an unexplained forever-queued row.
- Enforce exact-commit builds and store detected framework, serving mode, package manager, Node version, commands, and output path on each deployment for reproducibility.
- Fix cancel/dedup races with compare-and-set state transitions. Cancellation must end as `cancelled`, stop the container, emit one terminal event, and not send a failure notification.
- Implement rollback and promote through the same routing hook/reconciliation path as successful builds. Validate that the referenced artifact still exists before creating a ready record.
- Make route/database ordering failure-safe: a deployment is not externally `ready` until its immutable preview route is live; production activation is an explicit atomic/recoverable step.
- Persist the correct serving mode so SPA routes receive index fallback and static sites retain real 404 behavior.
- Make redeploy resolve the intended source semantics (same immutable commit versus latest branch head) and name the two operations distinctly.
- Handle duplicate commits per environment/branch correctly; a production push must not be suppressed merely because the same SHA previously produced a preview.
- Guarantee artifact immutability. Rollback/promote may share an artifact only while reference-aware GC protects it.
- Validate output content and size, prevent symlink/special-file extraction, and fail with actionable missing-output diagnostics.

**P1**

- Persist queue order and recover queued jobs without blocking startup. Bound enqueue latency and return a clear capacity error when saturated.
- Reconcile stale `queued/building/ready` states, orphan containers, incomplete artifact directories, and routes after crash/restart.
- Add per-phase timing and normalized failure categories (clone, install, build, output, route, timeout, OOM, cancel).
- Test lockfile cache invalidation and put an enforceable disk cap/retention policy on named Docker volumes.

**Exit criteria**

- Push, manual deploy, redeploy, cancel, dedup, preview, promote, rollback, restart recovery, and GC all pass state-machine integration tests.
- The commit and artifact served at every URL are provably the commit displayed in Hostbox.

### D. Finish GitHub as the primary experience

**P0**

- Persist webhook deliveries before acknowledging them; process via a bounded durable queue with delivery-ID uniqueness, attempts, last error, and terminal status.
- Return non-2xx when an event cannot be durably accepted. Retry transient GitHub/API failures with bounded backoff.
- Wire GitHub Deployment creation/status updates for queued, building, ready, failed, and cancelled states; persist/reuse `github_deploy_id` idempotently.
- Wire one updatable PR comment with build state, immutable preview URL, branch URL, commit, logs/dashboard link, and failure summary.
- Handle branch deletion, PR close/reopen/synchronize, installation suspend/delete, repository rename/transfer, and removed repository access.
- Remove immutable and branch-stable preview routes on close/delete according to the documented retention policy.
- Support fork PRs safely or reject them clearly; do not silently try to clone a fork branch from the base repository.
- Verify installation/repository association on every event and deployment, and define behavior for multiple projects using the same repository.

**P1**

- Surface webhook delivery health, last successful event, App permission problems, rate limiting, and circuit-breaker state in admin diagnostics.
- Add a “redeliver/retry” operation for failed stored events.
- Verify the GitHub App manifest permissions/events are the minimum needed and test upgrades to App configuration.

**Exit criteria**

- GitHub redelivery is idempotent, restart-safe, and produces exactly one Hostbox deployment/status/comment outcome.
- A real public GitHub repository and a private repository pass push and PR scenarios on the test VM.

### E. Finish routing, DNS, and TLS

**P0**

- Update routes immediately on build success, promote, rollback, domain verification/deletion, project rename/deletion, PR close, and branch deletion. Keep periodic full reconciliation as repair, not the primary user path.
- Add explicit branch-route removal and ensure full reconciliation cannot resurrect inactive previews.
- Verify custom domains against the expected instance target. Handle A/AAAA/CNAME, apex versus subdomain instructions, CNAME chains, IPv4/IPv6, propagation, and the server's configured public addresses.
- Do not activate a custom-domain route until verification succeeds. Remove it after the documented grace policy and notify the owner.
- Prove certificate issuance/renewal with no DNS provider (individual HTTP-01 names) and with every advertised wildcard DNS provider. Validate missing credentials at startup/setup.
- Test project/dashboard hostname collisions, branch slug collisions, long labels, wildcard coverage, IDN/punycode, mixed case, and reserved names.
- Preserve the last valid Caddy config when a dynamic update fails; expose the failure without making existing sites unavailable.

**P1**

- Show live DNS/TLS state and actionable checks in the dashboard instead of only “verified”.
- Define caching for hashed assets, HTML, SPA fallback, Brotli/gzip negotiation, security headers, custom 404 behavior, and immutable preview artifacts.
- Add redirect/canonical-host choices (`www`, apex, production alias) only if they fit the static-only v1 contract.

**Exit criteria**

- Preview, branch, production, dashboard, and custom-domain routes survive Caddy/Hostbox restart and certificate renewal tests.
- DNS pointing elsewhere cannot be marked verified.

### F. Complete framework/build support without becoming a PaaS

**P0**

- Create fixtures for every advertised framework and supported version. Each fixture must build in the same container path used in production and pass deep-link/static-asset checks.
- Make framework detection and overrides deterministic. Persist the result and show exactly what will run before/while building.
- Validate static-export requirements for Next.js, Nuxt, SvelteKit, Astro, and similar hybrid frameworks; reject server output.
- Decide and test npm, pnpm, Yarn, and Bun/Corepack versions. Respect `packageManager`, lockfiles, and frozen installs; avoid `npx --yes` fetching an unpinned Bun wrapper on every build.
- Define Node 18/20/22 lifecycle policy and prevent unsupported/arbitrary image tags.
- Correct monorepo root/output/cache behavior and handle multiple-app ambiguity in the creation UI.

**P1**

- Add image pre-pull/warmup and clear first-build progress without increasing idle processes.
- Bound clone size, artifact size, log size, build time, cache size, and queue depth, with per-instance configuration and useful errors.
- Decide whether plain static sites are copied directly or passed through a safer artifact staging step; apply the same path/symlink rules either way.
- Benchmark cold/warm builds and cache hit behavior on the minimum VM.

**Exit criteria**

- The published matrix is generated from green fixture tests; unsupported server output always fails safely and clearly.

### G. Repair dashboard, CLI, and accessibility

**P0**

- Fix profile/password routes and either implement session list/revoke endpoints or remove that UI from v1.
- Audit every dashboard hook and CLI client method against registered API routes, methods, bodies, response envelopes, auth, pagination, and errors. Add contract tests for both clients.
- Make `hostbox deploy` use the enqueueing endpoint; fix deployment lookup/logs/rollback and all response decoding.
- Fix dashboard lint errors and hook dependency warnings that can affect auth bootstrap.
- Prove first-run setup, refresh-cookie bootstrap, expired access refresh, logout, logout-all, route guards, and disabled registration.
- Make all partial/disabled/unavailable services visible: Docker unavailable, Caddy unavailable, GitHub unconfigured/suspended, build queue full, SMTP absent, and DNS pending.

**P1**

- Add consistent loading, empty, retry, offline, validation, destructive-confirmation, and partial-failure states.
- Make settings truthful: identify restart-required settings, validate them, enforce them, and show effective values rather than merely stored values.
- Add backup/create/restore/upgrade UI only after those operations are safe; otherwise link to tested operator commands.
- Test responsive layouts and keyboard navigation. Meet WCAG 2.1 AA basics: labels, focus, contrast, landmarks, status announcements, dialogs, tables, and reduced motion.
- Keep log rendering virtualized/bounded for large logs and sanitize ANSI/control sequences without turning logs into HTML.

**P2**

- Improve CLI completion, exit codes, TTY/non-TTY output, JSON stability, config permissions, token refresh/login expiry, version compatibility warnings, and shell install UX.
- Remove stale generated TypeScript/build files from source control and make builds reproducible from clean checkouts.

**Exit criteria**

- Dashboard browser tests and CLI integration tests cover every v1 command/screen against the real API.
- No visible control is a dead end and all clients pass the same authorization/ownership cases.

### H. Finish auth, secrets, and notifications

**P0**

- Resolve email scope. Either implement SMTP delivery for reset/verification and correct `email_verified` updates, or remove email-recovery/verification claims and ship a tested local admin reset command.
- Make setup single-winner/transactional so concurrent first-run requests cannot create multiple admins or leave `setup_complete` inconsistent.
- Normalize emails and enforce password policy/rate limits. Revoke all sessions after password/email/privilege changes as appropriate.
- Decide whether changing email requires current password and re-verification.
- Validate registration setting and project/user limits in the service layer, not just the UI.
- Keep environment and GitHub secrets encrypted with authenticated encryption and AAD; document encryption-key backup/rotation and the consequences of losing it.
- Never return secret values. Mask notification webhook URLs and treat them as secrets at rest and in logs/backups/UI.
- Validate notification event names and dispatch domain verified/unverified events from real domain transitions.

**P1**

- Add session listing/revocation with current-session identification if retained in v1.
- Add login/setup/password-reset audit events with retention/redaction rules.
- Add notification delivery observability, bounded concurrency, shutdown handling, and retry history rather than detached goroutines only.

**Exit criteria**

- Auth/session/recovery tests cover concurrency, expiry, replay, revocation, enumeration resistance, cookie flags, and restart.
- Secrets do not appear in API responses, URLs, logs, process arguments, or GitHub comments/statuses.

### I. Make data lifecycle, backup, restore, and upgrade safe

**P0**

- Make project deletion a service operation that stops builds and removes routes, artifacts, logs, clone temp data, and Docker cache volumes before/with metadata deletion. Make it retryable after partial failure.
- Make GC reference-aware for shared rollback/promote artifacts, current production, active branch URLs, and custom domains. Clean orphan project directories and cache volumes.
- Add disk high-water/critical-water behavior: refuse new builds before disk exhaustion, alert operators, and keep serving existing sites.
- Fix restore so it validates compatibility, quiesces writes/builds, checkpoints WAL, makes a safety backup, replaces the DB atomically, and actually restarts/reopens cleanly. Never continue using a closed DB handle.
- Define the backup unit. At minimum include SQLite plus required configuration/encryption key; clearly choose whether artifacts/Caddy state are backed up or reproducible.
- Test upgrade migrations from every supported v0/RC version, failed migration rollback, forward-only compatibility, and downgrade policy.
- Replace the non-existent Compose rollback behavior with pinned previous images/source revisions and a tested health-gated rollback.

**P1**

- Schedule automatic backups, expose last success/failure and retention, and support copying backups off the VM.
- Add integrity checks (`PRAGMA integrity_check`), WAL checkpoint policy, corruption recovery docs, and a restore drill.
- Ensure backup paths accepted by the API/CLI cannot escape the backup directory unless an explicit local-operator mechanism is used.

**Exit criteria**

- A fresh VM can restore a backup and serve the same projects/domains/secrets after a documented drill.
- Failed upgrades restore the previous healthy version without losing deployments created before the upgrade.

### J. Prove the lightweight promise

The lightweight claim is a v1 feature and must have a repeatable benchmark, not an estimate in `SPEC.md`.

**P0**

- Publish a minimum and recommended VM profile. Proposed gate: 1 vCPU, 512 MB RAM, and 10 GB disk can install, idle, serve the dashboard/sites, and build a small fixture with configured swap; recommend 1 GB+ for ordinary modern builds.
- On a clean supported VM, measure total steady-state RSS/PSS for Hostbox, Caddy, and their attributable Docker overhead after 30 minutes idle.
- Proposed idle gate: Hostbox + Caddy combined median RSS at or below 100 MB, total host memory at or below 300 MB, no swap growth, and idle CPU below 1% averaged over 15 minutes. Adjust only with published evidence.
- Measure cold start, restart recovery, dashboard latency, static-file throughput, SQLite size/WAL, scheduler wakeups, network chatter, and container/image/cache disk use.
- Reconcile build limits with host capacity. A 512 MB container limit on a 512 MB VM is not viable without swap/overcommit policy; detect available memory and fail early or select a safe limit.
- Ensure schedulers, metrics retention, Caddy sync, and rate-limiter cleanup do not cause leaks. Run 24-hour and seven-day idle/traffic soaks.

**P1**

- Provide a low-memory preset: one worker, smaller caches/logs/metrics retention, build admission control, and documented swap.
- Compare idle footprint and process count against current Coolify/Dokploy releases using the same VM methodology, while avoiding marketing claims that cannot be reproduced.
- Add regression budgets to a scheduled benchmark job or release checklist.

**Exit criteria**

- Benchmark scripts, raw results, Hostbox version, VM image, and configuration are committed/published.
- A v1 release candidate meets the declared idle and small-build gates without OOM, runaway CPU, or disk growth.

### K. Observability and honest administration

**P0**

- Split liveness from readiness. Readiness must report DB, Docker/build availability, Caddy reconciliation, disk pressure, and GitHub configuration health without exposing secrets.
- Correct disk metrics to use filesystem capacity/availability and include artifacts, logs, cache, backups, database/WAL, images, and volumes.
- Align all default/effective settings across migration defaults, environment config, admin API, dashboard, worker pool, GC, and docs.
- Make restart-required settings explicit or implement safe runtime reconfiguration; never display stored concurrency `N` while the pool still has a different worker count.
- Add request/build/deployment IDs consistently and actionable structured error categories.

**P1**

- Expose a small Prometheus/OpenMetrics endpoint or documented JSON metrics without adding a heavyweight observability stack.
- Add alerts for disk, memory/OOM, queue saturation, build failure rate, Caddy sync/TLS, backup age, webhook backlog, and notification failures.
- Bound metrics history and activity logs with retention/cleanup.
- Add a downloadable redacted diagnostics bundle for support.

**Exit criteria**

- An operator can distinguish “dashboard is alive” from “new deployments can succeed” and diagnose every primary dependency from one page/command.

### L. Installation, packaging, releases, and supportability

**P0**

- Choose the supported install model for v1. Recommended: pinned, signed/checksummed release images in Compose; retain source builds as a contributor/development path.
- Fix standalone API packaging by using a CGO-capable build per target (or changing SQLite driver deliberately), then run each archive on its target architecture in CI.
- Fix CLI archive contents/installer naming and verify checksums before installation.
- Pin image versions/digests instead of mutable `latest`; align GHCR owner/image names and version labels.
- Test the installer on every supported distro/architecture, including no Docker, non-default Docker GID, occupied ports, missing DNS, rerun, interrupted install, and uninstall.
- Do not silently continue when Compose installation fails. Validate Docker Compose, OpenSSL, firewall/ports, disk, RAM/swap, DNS provider credentials, architecture, and required paths before mutation.
- Make install/update non-destructive to `.env` and data. Take/verify a backup before upgrade and health-gate the new version.
- Remove the tracked local `api` binary and generated build metadata; add ignore/release hygiene checks.
- Generate an SBOM, checksums, provenance/attestations, changelog, migration notes, and a release-candidate channel.

**P1**

- Add a tested uninstall procedure with explicit keep/delete-data choices.
- Add compatibility checks between CLI and server versions.
- Define supported-version/security-fix policy and a vulnerability-reporting channel.
- Make release notes distinguish breaking config/schema changes and required operator action.

**Exit criteria**

- A clean VM install, upgrade, rollback, CLI install, backup, restore, and uninstall are scripted and pass from published artifacts—not from an uncommitted checkout.

### M. Testing, CI, and release evidence

**P0**

- Restore a completely green baseline: Go test/vet/race/build, dashboard lint/type/build, marketing clean install/build, Docker/Compose validation, and vulnerability policy.
- Add API integration tests for every route, ownership/admin boundary, response shape, pagination/filtering, and failure state.
- Add dashboard browser tests for setup, login/refresh, GitHub connection (mocked contract), project create/edit/delete, deploy/log/cancel, env vars, domains, notifications, profile, admin, and logout.
- Add CLI tests against a real test server for every command and JSON output.
- Add build fixtures and real Docker integration tests for all supported frameworks/package managers, path isolation, exact commits, timeout/OOM/cancel, cache, artifacts, and logs.
- Add Caddy integration tests that request real hosts and verify SPA fallback, static 404, compression, headers, route activation/removal, and restart reconciliation.
- Add a disposable public-VM acceptance job/runbook for GitHub webhooks, wildcard/individual TLS, custom domains, install, push, PR, promote, rollback, upgrade, and restore.
- Add negative/chaos tests: restart Hostbox/Caddy/Docker during builds, GitHub outage/rate limit, DNS failure, full disk, corrupt backup, unavailable registry, and network interruption.

**P1**

- Enforce coverage on critical packages/flows rather than a vanity repository percentage.
- Add migration fixtures for old databases and test both amd64/arm64 release artifacts.
- Add performance/soak suites for idle resources, concurrent API use, log streams, repeated webhooks, and long-running uptime.
- Publish the release evidence for each RC: commit, artifacts, test results, VM scenarios, benchmark, known limitations, and rollback result.

**Exit criteria**

- No required v1 behavior relies only on a manual claim or unit mock.
- CI is required on protected `main`, and releases can only be created from a green, reviewed, signed tag.

### N. Documentation and launch readiness

**P0**

- Reconcile `README.md`, `SPEC.md`, `ARCHITECTURE.md`, `ONBOARDING.md`, self-hosting docs, phase plans, CLI help, dashboard copy, and marketing against actual v1 behavior.
- Mark old phase plans as historical/aspirational so they are not mistaken for implemented state.
- Document DNS patterns, GitHub App creation/permissions, supported frameworks, static-export recipes, environment scopes, resource sizing, backups/restores, upgrades/rollback, and security boundaries.
- Add a troubleshooting decision tree with stable error codes for install, GitHub, clone, build, output, Caddy, DNS, TLS, OOM, disk, auth, and restore failures.
- Add a v1 operator checklist and a prominent “not SSR / not arbitrary containers” statement.

**P1**

- Add contributor setup with one canonical toolchain version and clean-checkout commands.
- Add architecture decision records for static-only scope, SQLite, Caddy, Docker socket, webhook durability, and the resource budget.
- Prepare changelog, v1 migration guide, known limitations, security policy, support matrix, and rollback instructions.

**Exit criteria**

- A new operator can install and complete the golden path using only published documentation.

## 6. Recommended execution sequence

### Milestone 0 — Truthful baseline

1. Freeze v1 scope/support matrix and API contract.
2. Make all existing checks green; add marketing to CI.
3. Remove tracked binaries/generated artifacts and fix release/CLI packaging.
4. Convert the findings in this document into independently reviewable issues with owners and acceptance tests.

### Milestone 1 — Secure deterministic core

1. Fix path/symlink isolation, exact-commit checkout, credentials, SSRF, and stream-token exposure.
2. Replace duplicate deployment creation paths with the enforced state machine.
3. Fix framework/serving-mode persistence, cancellation, rollback, promote, cleanup, and recovery.
4. Add Docker/build/Caddy integration fixtures before expanding features.

### Milestone 2 — Complete the GitHub-to-URL loop

1. Add durable webhook intake and idempotency.
2. Wire statuses/comments and all lifecycle events.
3. Complete immediate Caddy route/domain/TLS lifecycle.
4. Run the first real public-VM push/PR acceptance flow.

### Milestone 3 — Complete clients and operations

1. Repair dashboard/CLI contracts and add E2E coverage.
2. Make settings, diagnostics, limits, disk admission, notifications, and auth recovery honest and complete.
3. Finish project deletion, backup/restore, upgrade/rollback, and clean-VM installers.

### Milestone 4 — Lightweight release candidate

1. Run resource benchmarks, long idle soak, build soak, failure injection, security scans, and restore drills.
2. Fix regressions without adding always-on infrastructure.
3. Reconcile all docs/marketing and publish `v1.0.0-rc.1` with known limitations.

### Milestone 5 — v1.0.0

1. Require at least one RC upgrade and rollback on the test VM.
2. Require the full golden-path acceptance matrix on amd64 and arm64 where supported.
3. Tag v1 only after every gate below has evidence and no open P0 issue.

## 7. v1 release gates

### Golden path

- [ ] Clean supported VM installs from a pinned release artifact.
- [ ] Browser setup creates exactly one admin and survives restart.
- [ ] GitHub App connects and lists public/private repositories.
- [ ] Push builds the exact SHA and serves a production URL over HTTPS.
- [ ] PR creates an isolated preview, GitHub status, and one updated comment.
- [ ] Logs stream and remain readable after completion/restart.
- [ ] Client-side SPA deep links and static-site 404s behave correctly.
- [ ] Custom domain verifies only against the instance and receives a valid certificate.
- [ ] Cancel, dedup, promote, rollback, PR close, and branch delete produce correct durable state/routes.

### Safety and recovery

- [ ] No known open P0 security issue or unfixed dependency finding above the release policy.
- [ ] Exact-commit, path traversal, symlink, archive, SSRF, auth, ownership, and secret-redaction tests pass.
- [ ] Full disk, OOM, registry/GitHub/DNS/Caddy outage, and service restart fail safely.
- [ ] Backup/restore on a new VM and upgrade/rollback of the previous RC both pass.
- [ ] Project deletion and GC leave no active route or unbounded orphan data.

### Quality and footprint

- [ ] Go race/test/vet/build, frontend lint/type/build/E2E, marketing build, Docker builds, release archives, and scans are green.
- [ ] Every advertised framework fixture passes from a clean cache.
- [ ] amd64 and arm64 artifacts are installed/executed in CI or release acceptance.
- [ ] The documented 512 MB/minimum-VM scenario passes, and measured idle CPU/RAM/disk results are published.
- [ ] A seven-day soak shows no material memory, goroutine, DB/WAL, log, cache, route, or container leak.

### Documentation and release

- [ ] Docs/CLI/UI/marketing match the actual support matrix and limitations.
- [ ] SBOM, checksums, provenance, changelog, migration notes, security policy, and rollback instructions ship with v1.
- [ ] `main` is protected and the release tag points to the reviewed green commit.

## 8. Definition of done for every roadmap item

An item is complete only when:

1. The behavior and failure behavior are defined.
2. The implementation uses the shared service/domain path rather than a UI-only workaround.
3. Authorization, security, resource, migration, and cleanup effects are considered.
4. Automated positive and negative tests exist at the lowest useful level plus an integration/E2E test when user-visible.
5. Metrics/logs/errors make failures diagnosable without exposing secrets.
6. Operator/user documentation and client contracts are updated.
7. The relevant release gate has reproducible evidence.

## 9. Post-v1 backlog

Only consider these after the static single-node v1 gates are stable:

- GitLab/Bitbucket providers.
- Remote/S3-compatible artifacts and off-host builds.
- Multi-node workers/high availability.
- Organizations, team roles, audit export, SSO.
- Server runtimes/functions/SSR.
- Arbitrary containers, databases, and broader PaaS capabilities.

Those features would materially change Hostbox's security model and idle footprint. They should not delay or dilute the focused v1.
