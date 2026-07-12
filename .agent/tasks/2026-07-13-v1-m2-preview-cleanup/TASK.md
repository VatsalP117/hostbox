# Task

Complete preview route cleanup for pull-request closure and branch-deletion GitHub events.

Add a Caddy branch-route removal operation that accepts a project ID and raw branch name, extend the GitHub route-removal boundary, and ensure both event handlers deactivate associated preview deployments and remove immutable plus branch-stable routes. Cleanup must attempt all possible operations and return failures to durable webhook processing instead of logging false success.

## Scope

- `internal/services/caddy/manager.go`
- `internal/services/caddy/manager_test.go`
- `internal/services/github/interfaces.go`
- `internal/services/github/push_handler.go`
- `internal/services/github/pr_handler.go`
- `internal/services/github/events_test.go`
- This task artifact directory

The orchestrator owns runtime/main wiring and all other milestone-two components.

## Acceptance Criteria

- Raw branch names, including slash-delimited names, resolve to the same stable-route ID used at creation.
- PR close deactivates branch deployments, attempts every returned immutable-route removal, and removes the stable branch route.
- Deleted branch pushes perform the same cleanup after resolving the project.
- Cleanup failures are returned and preserve all constituent errors while other possible cleanup operations are still attempted.
- Non-branch refs perform no cleanup; ordinary push deployment behavior remains unchanged.
