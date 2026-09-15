# Bounded coordinator — Go

The coordinator reads a bounded source batch, fans out one short-lived child
orchestration per item, **waits for every child**, and uses `ContinueAsNew` before
reading the next batch. Only a cursor, batch number, and processed count cross
the reset boundary.

The demo processes **three batches of five tenant-scoped changes**. Source
reads and applying changes are explicitly **simulated**, stateless activities;
no tenant resources are modified. Child IDs include the parent ID and item ID,
so different batches never reuse child instances.

Fixture cursors advance by whole five-item pages. The source rejects requested
bounds below five rather than silently skipping the rest of a page; larger
bounds (up to 50) still return at most five items.

## Prerequisites

- Go **1.25 or newer**, Docker, and a running Durable Task Scheduler emulator.
- Follow [shared emulator and live Azure setup](../README.md).
- The shared module pins SDK `v1.0.0-beta.1`.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./bounded-coordinator`.
Worker and client run together. Normal execution finishes in under a minute;
`-timeout` defaults to two minutes.

## Real continuation and event-carryover verification

The bounded demo pauses at a verification checkpoint **after** each child batch
has finished. Its client reads actual scheduler history and verifies:

1. Three **different execution IDs** for the same coordinator instance.
2. Each execution contains exactly one batch activity and five completed
   children, with exact tenant payloads and `processed:item-N-M` receipts.
3. Each new execution's persisted input has the previous batch's compact state.
4. An event sent during execution one appears in history **before** the first
   reset, survives both resets, and is consumed only in execution three.

The client then acknowledges each checkpoint so processing continues. These are
real continuations, not a counter inside one unbounded orchestration.
`task.WithKeepUnprocessedEvents()` is essential: removing it fails the carryover
checks. History API errors or missing execution IDs fail the sample; checks are
not skipped.

Checkpoints have a 15-second durable safety timeout. On an error, cleanup targets
only this run's coordinator and its own children. All activity/child work is
finished before continuation or normal shutdown.

## Expected output

Three evidence records have distinct `execution_id` values, each showing:

```json
{"batch_activities": 1, "completed_children": 5, "carryover_events": 1}
```

The final JSON result includes:

```json
{
  "total_batches": 3,
  "processed": 15,
  "completed": true,
  "carryover": "queued-before-first-history-reset"
}
```

```text
SAMPLE_OK bounded-coordinator
```

History is not purged. Open <http://localhost:8082> to inspect the coordinator's
latest, small execution and all 15 completed children. All registrations begin
with `GoBoundedCoordinator`, with automatic worker filters.

## Production adaptation

Replace the finite source fixture with a queue/database cursor and idempotent
tenant-change activities. Remove the **demo-only client verification gates and
three-batch stop condition**, not the `WhenAll` barrier or the history reset.
Keep state compact and preserve unconsumed external events across every
continuation. Never continue as new while child work is outstanding.

## Unit tests

```bash
go test -mod=readonly .
```

Tests check cursor determinism, bounds, exact tenant changes, exhausted input,
invalid carry-forward state, and rejection of incorrect history/child/carryover
evidence. They do not connect to a scheduler.
