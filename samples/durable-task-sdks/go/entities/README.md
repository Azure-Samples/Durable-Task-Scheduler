# Durable entities (Go)

## Description

This sample demonstrates persisted counter state, client signals, orchestration
signals and calls, and a scheduled reset. Each invocation owns fresh
`go-entities-*` instance/entity keys.
The worker uses registration-derived work-item filters.

Client signals produce `100 - 25 = 75`. A separate orchestration signals
`10 + 5 - 3`, reads `12`, and schedules a reset five seconds into the future using
the orchestration's deterministic clock. It verifies that the later value is `0`,
the earlier read preceded the due time, and the reset's **actual entity operation
timestamp** was not earlier than that due time. A final client read verifies the
persisted state rather than assuming that sending a signal means it was handled.

## Prerequisites

- Go 1.25.0 or later, using the shared module's pinned
  `github.com/microsoft/durabletask-go v1.0.0-beta.1`.
- An existing DTS emulator task hub or an existing Azure task hub with data-plane
  access. Follow the [shared emulator/live authentication setup](../README.md).
  No additional Azure resources are needed.

## Run

From this directory:

```bash
go run .
```

The default deadline is two minutes; `go run . -timeout 3m` changes that bound.
Tests need no scheduler:

```bash
go test -mod=readonly .
```

## Expected result

The command asserts completion, arithmetic, timing, and persisted state before
printing:

```text
Direct signals: 100 - 25 = 75
Orchestration signals and calls: 10 + 5 - 3 = 12; scheduled reset = 0
```

The JSON result contains `before: 12`, `after: 0`, and UTC `read_at`, `due_at`,
and `reset_at` timestamps satisfying `read_at < due_at <= reset_at`. The final
line is exactly:

```text
SAMPLE_OK entities
```

Timeouts, early delivery, missing resets, or wrong results fail the command.
Completed orchestration history and the two owned entity states remain available
for inspection; the demo does not query or delete other users' entities.

## How it works

- One process hosts the worker and bounded client.
- `task.WithSignalEntityScheduledTime` schedules future signals;
  `CurrentTimeUtc` and durable timers keep orchestration code replay-safe.
- The entity stores `{value, reset_at}` so the demo can verify delivery time.
  `get` returns an integer; `snapshot` and `delete` support state inspection
  and removal.
- The demo polls durable/server state with bounded waits; it never treats a
  fixed sleep or an accepted signal as proof of success.
