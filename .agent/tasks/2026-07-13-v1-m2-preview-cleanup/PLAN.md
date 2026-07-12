# Implementation Plan

## 1. Task Summary

Close the route-lifecycle gap for closed pull requests and deleted branches.

## 2. Current System Understanding

Immutable preview routes can be removed individually, but stable branch routes have no removal method. PR close logs and suppresses immutable-route failures, while deleted branch pushes are ignored before project resolution.

## 3. Scope

### In Scope

- Caddy stable branch-route deletion by raw branch name.
- GitHub handler cleanup for PR close and branch deletion.
- Aggregated cleanup errors and focused tests.

### Out of Scope

- Runtime/main wiring, database/repository changes, webhook queue processing, deployment status feedback, migrations, and production reconciliation.

## 4. Proposed Technical Approach

Derive the stable route ID with Caddy's existing `Slugify` convention. Add that operation to the handler boundary. In both cleanup paths, deactivate first, remove each immutable route returned by deactivation, always attempt stable-route deletion, and return `errors.Join` over contextualized failures.

## 5. Step-by-Step Execution Plan

1. Add and test `RemoveBranchRoute` in the Caddy manager.
2. Extend `RouteRemover` and inject it into the push handler.
3. Implement deleted-branch cleanup after branch-ref validation and project resolution.
4. Upgrade PR-close cleanup to remove the stable route and return route errors.
5. Add event tests for slash branches, both event types, error aggregation, and ignored non-branch refs.
6. Run focused tests, full Go tests, race checks, vet, and builds in coordination with milestone integration.

## 6. Test Plan

- Verify raw `feature/my change` maps to `route-branch-<project>-feature-my-change`.
- Verify PR close removes all immutable routes and the raw branch's stable route.
- Verify deleted `refs/heads/feature/test` performs equivalent cleanup.
- Verify multiple route/deactivation errors are returned and independent cleanup is attempted.
- Verify deleted tag refs do not touch nil dependencies.
- Verify ordinary push creation tests remain green.

## 7. Acceptance Criteria

All criteria in `TASK.md` pass through focused automated tests, with no files changed outside assigned ownership.

## 8. Risks and Guardrails

- Preserve raw branch names at the GitHub boundary; normalize only inside Caddy.
- Do not stop after the first route error.
- Do not infer a branch from tag or other refs.
- Do not alter normal push eligibility or deployment creation behavior.

## 9. Executor Instructions

Keep constructor changes local to GitHub handlers and notify the orchestrator of required main wiring. Do not commit independently.
