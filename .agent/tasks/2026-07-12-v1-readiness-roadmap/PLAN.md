# Implementation Plan

## 1. Task Summary

Assess the implementation rather than treating the aspirational specification as complete, then produce a practical roadmap from the current v0 state to v1.

## 2. Current System Understanding

Hostbox is a Go/Echo monolith with an embedded React dashboard, SQLite, an in-process Docker build worker, Caddy routing, a GitHub App integration, a Go CLI, install/update scripts, and release workflows. Existing documentation includes an aspirational v1 specification and phase plans; the audit must distinguish those documents from verified behavior.

## 3. Scope

### In Scope

- Backend, dashboard, CLI, build isolation, GitHub, Caddy/TLS, persistence, packaging, operations, security, reliability, resource footprint, tests, docs, and release process.
- Verification of current automated checks.
- A phased roadmap with P0/P1/P2 priorities and measurable v1 gates.

### Out of Scope

- Implementing the roadmap.
- Production infrastructure changes, schema changes, dependency changes, or release publication.
- Expanding Hostbox into a general-purpose PaaS or SSR/container host.

## 4. Proposed Technical Approach

Compare the intended product contract in `SPEC.md` and existing docs with actual routes, services, worker behavior, UI calls, CLI commands, scripts, Compose/Docker configuration, tests, and CI. Record only gaps supported by repository evidence, separating launch blockers from improvements.

## 5. Step-by-Step Execution Plan

1. Inventory repository structure, history, documentation, and current worktree.
2. Trace end-to-end product flows and enumerate incomplete/stubbed behavior.
3. Inspect security, failure recovery, resource limits, installation/upgrades, backups, and observability.
4. Run backend, frontend, build, and lightweight footprint checks available locally.
5. Write the v1 roadmap with baseline, definition of v1, prioritized workstreams, sequencing, gates, and deferred scope.
6. Review the document against findings and commit the documentation artifacts.

## 6. Test Plan

- `go test ./...`
- `go vet ./...`
- `go build ./cmd/api ./cmd/cli`
- `npm run lint`, TypeScript check, and production build in `web/`
- Production Compose configuration validation when Docker is available.
- Documentation cross-checks against routes, CLI commands, configuration, and tracked files.

## 7. Acceptance Criteria

- The roadmap clearly states what works, what is partial, and what remains.
- Each v1 requirement has a concrete outcome and verification gate.
- Lightweight resource goals are measurable and treated as a release gate.
- Non-goals keep v1 frontend/static-only and single-node.
- The new documentation is committed on the task branch, with no unrelated user files included.

## 8. Risks and Guardrails

- Existing planning docs may overstate implemented behavior; source and tests take precedence.
- Local checks may not exercise real GitHub, DNS, ACME, or clean-VM behavior; these become explicit VM/E2E gates.
- Preserve `.opencodeignore` and any other user changes.

## 9. Executor Instructions

Make documentation-only changes. Cite paths and concrete behaviors where helpful, avoid inventing deadlines, and do not label v1 production-ready until all release gates are satisfied.
