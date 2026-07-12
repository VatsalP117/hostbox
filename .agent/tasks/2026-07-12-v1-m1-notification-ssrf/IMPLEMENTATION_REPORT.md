# Implementation Report

## Summary

Enforced the existing webhook URL security policy before notification configuration persistence and before every notification service send path. Handler failures use the existing validation error response type with an actionable `Invalid webhook URL` message. Dispatch now also refuses unsafe legacy or directly inserted database records before scheduling a client call.

Focused tests cover insecure HTTP, IPv4 loopback, private IPv4, IPv6 loopback, update non-mutation, manual test rejection, unsafe dispatch rejection, and valid HTTPS acceptance. All address cases use literal IPs; valid cases use RFC 5737 TEST-NET-2 (`198.51.100.10`) with injected recording clients, avoiding public DNS and outbound network dependencies.

## Files Changed

- `internal/api/handlers/notifications.go`: validates create, update, and manual test webhook URLs.
- `internal/api/handlers/notifications_test.go`: adds focused handler persistence, mutation, and test-send boundary coverage.
- `internal/services/notification/service.go`: validates stored URLs before async dispatch and manual service sends.
- `internal/services/notification/notification_test.go`: updates dispatch tests to deterministic valid HTTPS test clients and adds unsafe send-boundary coverage.
- `.agent/tasks/2026-07-12-v1-m1-notification-ssrf/TASK.md`: records task scope.
- `.agent/tasks/2026-07-12-v1-m1-notification-ssrf/PLAN.md`: records implementation and test plan.
- `.agent/tasks/2026-07-12-v1-m1-notification-ssrf/IMPLEMENTATION_REPORT.md`: records implementation results.

## Commands Run

- `gofmt -w internal/api/handlers/notifications.go internal/api/handlers/notifications_test.go internal/services/notification/service.go internal/services/notification/notification_test.go`
- `go test ./internal/api/handlers ./internal/services/notification ./internal/platform/sanitize`
- `go test ./internal/api/handlers -run 'TestNotification' -count=1`
- `go test ./internal/services/notification -run 'TestService_(Dispatch|SendTest)' -count=1 -race`
- `go test ./cmd/... ./internal/... ./migrations`
- `go vet ./cmd/... ./internal/... ./migrations`
- `go build ./cmd/api ./cmd/cli`
- `go test -race ./internal/api/handlers ./internal/services/notification`

## Tests

- Focused handler notification tests: passed.
- Focused notification service tests under race detector: passed.
- Full command/internal/migrations Go test suite: passed.
- Full focused handler/service race run: passed. The macOS linker emitted a non-failing SQLite `LC_DYSYMTAB` warning for both test binaries.
- Go vet: passed.
- API and CLI builds: passed.

The first combined focused run briefly failed to compile the handler package because another parallel track had an in-progress unused import in `internal/worker/executor.go`. No notification change was needed; the focused and full checks passed after that parallel edit completed.

## Deviations From Plan

None. Defense-in-depth dispatch validation was included as planned so stored unsafe configurations cannot bypass handler validation.

## Known Risks

- Validation inherits the existing sanitizer's DNS policy: unresolvable hostnames are allowed, and validation does not pin the resolved address used by the later HTTP connection. Addressing DNS failure policy and DNS rebinding requires a separate sanitizer/transport-hardening task and was outside this track's ownership.
- This track does not redesign retries or make delivery durable.

## Next Steps

- Review this track together with the parallel sanitizer and worker changes because the notification boundary intentionally consumes their shared `sanitize.ValidateWebhookURL` contract.
- Consider a later SSRF-hardened HTTP transport that validates and pins dial targets if the v1 threat model requires DNS-rebinding resistance.
