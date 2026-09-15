# Eternal orchestrations — Go

Run a periodic cleanup activity, await a durable timer, and **continue as new**
with a compact counter and accumulated removal count. The instance ID remains
the same while its execution history is replaced.

Like Python, the demo stops after **five cycles**. It uses 250 ms intervals
instead of 15-second timers plus five-second activity sleeps. Cleanup is an
explicit **in-memory simulation**: each cycle identifies two expired records and
retains one current record. No user files, database rows, or scheduler instances
are deleted.

## Prerequisites

- Go **1.25 or newer**, Docker, and a running Durable Task Scheduler emulator.
- See [shared emulator and live Azure setup](../README.md).
- The parent module pins Durable Task Go SDK `v1.0.0-beta.1`.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./eternal-orchestrations`.
The worker and client run together. The client waits through all continuations,
asserts the exact result, and reads the latest execution's history to verify
that it contains **only cycle five**, one cleanup activity, and one fired timer.
An unavailable history API is an error, not a skipped check.

Normal execution takes a few seconds; the outer `-timeout` defaults to two
minutes. No recurring work remains when the process exits.

## Expected output

The JSON output includes the instance ID, final execution ID,
`latest_cleanup_activities: 1`, and:

```json
{"iterations": 5, "total_removed": 10, "last_message": "Cleanup completed"}
```

```text
SAMPLE_OK eternal-orchestrations
```

Inspect the retained latest execution at <http://localhost:8082>. History is
reset by continuation, **not** by a purge command. Registered names start with
`GoEternal`, and automatic worker filters isolate the sample.

## Production continuation

For a genuinely eternal workflow, replace the finite fixture and its five-cycle
stop condition with a real cleanup source and operational stop policy. Keep only
compact state across executions; do not carry an ever-growing list of receipts.
The code already uses `task.WithKeepUnprocessedEvents()` so future external
control events are not discarded at continuation boundaries. Finish activities,
timers, and any child work before resetting history. Real cleanup activities must
be idempotent under at-least-once execution.

## Unit tests

```bash
go test -mod=readonly .
```

Tests cover fixture partitioning, exact receipts, invalid state, and rejection of
history that has not actually reset. These tests do not connect to a scheduler.
