# Implementation Plan

## 1. Task Summary

Build a small lifecycle reporter that translates Hostbox deployment states into GitHub Deployment statuses and a single preview PR comment.

## 2. Current System Understanding

Typed GitHub clients and unused status/comment helpers already exist. Deployments already contain a nullable `github_deploy_id`, but the repository lacks a narrow set-if-unset method. The GitHub runtime can be configured dynamically, so the reporter cannot retain a startup-only concrete client.

## 3. Scope

### In Scope

- Small client/provider/store interfaces for lifecycle feedback.
- Repository parsing and required installation metadata validation.
- GitHub Deployment creation, ID persistence, and later status updates.
- Marker-based preview PR comment create/update behavior for every lifecycle state.
- Dashboard/log and deployed-environment URLs.
- Focused GitHub and repository tests.

### Out of Scope

- Runtime, worker, deployment service, event handler, router, or application wiring.
- Database migrations or schema changes.
- Webhook durability and preview-route cleanup.

## 4. Proposed Technical Approach

Expose a synchronous `LifecycleReporter.Report` method accepting the current project and deployment. Resolve a client on each call through a provider. Ignore projects with no GitHub metadata, reject partial or malformed metadata, serialize feedback creation in-process, create/update GitHub status, persist a newly created ID with set-if-unset semantics, then create or update the marked PR comment for preview PR deployments.

## 5. Step-by-Step Execution Plan

1. Replace concrete helper client dependencies with narrow interfaces.
2. Add metadata validation and queued/cancelled comment rendering.
3. Add atomic repository persistence for `github_deploy_id`.
4. Add the dynamic lifecycle reporter and URL/duration construction.
5. Add focused behavior and persistence tests.
6. Run focused tests, full Go tests, formatting, and diff checks.

## 6. Test Plan

- Prove one GitHub Deployment is created and persisted across queued/ready updates.
- Prove statuses reuse the persisted ID and carry correct state and URLs.
- Prove the marker comment is created once then updated.
- Prove fully disconnected projects are safe no-ops without requesting a client.
- Prove malformed repositories and missing installation IDs fail before API calls.
- Prove repository set-if-unset preserves the original ID.

## 7. Acceptance Criteria

- One persisted GitHub Deployment ID is reused across lifecycle updates.
- Preview PR feedback maintains one marked comment.
- Production deployments do not create PR comments.
- URLs point to the environment and Hostbox deployment dashboard.
- Errors are returned synchronously.
- Focused tests pass and no files outside assigned scope/task artifacts are edited.

## 8. Risks and Guardrails

- Do not treat optional GitHub integration as mandatory for manual projects.
- Reject incomplete metadata rather than target an ambiguous API path.
- Persist a created ID even when its initial status request fails, preventing a retry from creating another deployment.
- Do not detach reporting into goroutines.

## 9. Executor Instructions

Stay within the assigned files, make no runtime/application integration changes, do not commit, and report unrelated full-suite failures separately.
