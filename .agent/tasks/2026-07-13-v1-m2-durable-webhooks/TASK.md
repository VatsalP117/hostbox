# Task

Implement durable and idempotent GitHub webhook intake for the v1 milestone-two GitHub integration slice.

The implementation owns the additive webhook-delivery migration and model, repository persistence and atomic state transitions, a bounded in-process delivery processor, and the HTTP webhook acceptance path. Runtime/main wiring remains with the milestone orchestrator.

## Required Outcomes

- Verify GitHub signatures before storing deliveries.
- Bound request payload size and require delivery/event headers.
- Persist a unique GitHub delivery ID before returning HTTP 202.
- Treat duplicate delivery IDs as successful idempotent acceptance.
- Atomically claim queued deliveries, recover interrupted processing rows, and retry with bounded attempts/backoff.
- Use fixed workers, bounded scans, and shutdown-aware coordination without per-delivery goroutines.
