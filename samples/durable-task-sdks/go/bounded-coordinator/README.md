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
`-timeout` defaults to two minutes and accepts `-timeout 3m`.
The client schedules one coordinator, waits for the three batches, and prints
its result. There are no verification handshakes or history reads in the demo.
All children finish before continuation or normal shutdown; error cleanup
targets only this run's coordinator and its own children.

## Expected output

JSON output contains a unique coordinator instance ID and:

```json
{
  "total_batches": 3,
  "processed": 15,
  "completed": true
}
```

History is not purged. Open <http://localhost:8082> to inspect the coordinator's
latest, small execution and all 15 completed children. All registrations begin
with `GoBoundedCoordinator`, with automatic worker filters.

## Production adaptation

Replace the finite source fixture with a queue/database cursor and idempotent
tenant-change activities. Replace the three-batch source limit, not the `WhenAll`
barrier or the history reset.
Keep state compact and preserve unconsumed external events across every
continuation. Never continue as new while child work is outstanding.

## Code map

Read [workflow.go](workflow.go): read a batch, start children, await all, and
continue as new. [activities.go](activities.go) contains the bounded cursor
source and simulated changes. [client.go](client.go) runs one coordinator;
[worker.go](worker.go) registers tasks; [main.go](main.go) starts the CLI.

## Tests

```bash
go test .
```

Tests check cursor determinism, bounds, exact tenant changes, exhausted input,
invalid carry-forward state, and rejection of incorrect history/child/carryover
evidence. They do not connect to a scheduler. Full backend verification is
explicitly opt-in:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) wraps the same coordinator in a
test-only observer. The SDK commits completion/continuation when the registered
root returns, so the observer can pause after the real workflow finishes a
batch without adding hooks to production code. Its checkpoints have a
15-second safety timeout.

The test verifies three distinct execution IDs, one batch activity and five
exact child results per execution, compact carry-forward inputs, and an event
observed before the first reset that survives both resets and is consumed at
the end. Missing history APIs, missing events, or unchanged execution IDs fail
the test; no checks are skipped after opt-in.
