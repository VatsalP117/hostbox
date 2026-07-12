# Implementation Plan

## 1. Task Summary

Apply `sanitize.ValidateWebhookURL` at notification persistence and outbound-send boundaries, with focused regression coverage.

## 2. Current System Understanding

Notification create and update handlers currently persist any URL accepted by DTO syntax validation. Manual test sends and asynchronous dispatches pass stored URLs directly to channel clients. The platform sanitizer already enforces HTTPS and rejects resolved private IPv4 and IPv6 loopback targets, but notification code does not call it.

## 3. Scope

### In Scope

- Validate webhook URLs before create and update persistence.
- Validate URLs before manual and asynchronous service sends.
- Return safe, actionable validation errors without changing response envelopes.
- Add focused handler and service tests for insecure, loopback/private, IPv6 loopback, and valid HTTPS targets.

### Out of Scope

- Changes to sanitizer behavior or DNS resolution.
- Database schema, public routes, DTO shapes, deployment infrastructure, or notification delivery redesign.

## 4. Proposed Technical Approach

Use a small handler helper that converts sanitizer failures to the existing bad-request application error. Add defense-in-depth checks in the notification service immediately before selecting/scheduling a send. Tests use literal addresses so they do not depend on public DNS: private/loopback literals for rejection and an RFC documentation-range public IPv4 literal for acceptance.

## 5. Step-by-Step Execution Plan

1. Add handler URL checks after DTO validation and before model mutation or repository writes.
2. Check the stored URL in the manual test handler before constructing or sending its payload.
3. Check URLs in dispatch and `SendTest` before any client call.
4. Add focused handler persistence/non-mutation tests and service no-send/send tests.
5. Run formatting, focused tests, full Go tests, vet, and build checks.

## 6. Test Plan

- Create rejects HTTP, loopback IPv4, private IPv4, and IPv6 loopback without persistence.
- Create accepts deterministic valid HTTPS using a documentation-range public IP.
- Update rejects an unsafe replacement without changing the stored URL.
- Manual test handler rejects an unsafe stored URL before network activity.
- Notification service blocks unsafe dispatch/test-send URLs and permits a valid HTTPS URL to reach an injected test client.

## 7. Acceptance Criteria

- Unsafe targets cannot be newly persisted or used by notification sends.
- Validation failures are HTTP 400 at handler boundaries and retain the existing error type/shape.
- Valid HTTPS notification configuration behavior remains intact.
- Focused and repository-wide Go checks pass.

## 8. Risks and Guardrails

- Do not alter sanitizer semantics in this track.
- Do not expose webhook secrets in error messages.
- Do not introduce new dependencies or public API changes.
- Keep edits limited to notification handler/service code, tests, and this task folder.

## 9. Executor Instructions

Implement the narrow checks and tests described above. Do not commit independently. Report any sanitizer limitation as a known risk rather than broadening scope.
