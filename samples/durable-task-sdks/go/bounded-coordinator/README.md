# Bounded coordinator (Go)

The coordinator reads a batch with a size limit and starts one child workflow
for each item. It **waits for every child**, then calls `ContinueAsNew` before
reading the next batch. It carries only the batch number, processed count, and
a cursor that marks the next position in the source.

The demo processes **three batches of five changes for tenants**. A tenant
represents a customer or organization. Reading and applying these changes are
**simulated** activities; they do not change tenant resources or keep state.
Child IDs include the parent ID and item ID,
so different batches never reuse child instances.

The sample cursor moves forward by five items at a time. The source rejects
batch limits below five so that no part of a page is skipped. Larger limits,
up to 50, still return at most five items.

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
its result. The demo does not read history or pause for test checks.
All children finish before the next execution or normal shutdown. If an error
occurs, cleanup affects only this run's coordinator and children.

## Expected output

JSON output contains a unique coordinator instance ID and:

```json
{
  "total_batches": 3,
  "processed": 15,
  "completed": true
}
```

History is not deleted. Open <http://localhost:8082> to view the coordinator's
latest execution and all 15 completed children. All registered names begin
with `GoBoundedCoordinator`, with automatic worker filters.

## Production adaptation

Replace the sample source with a queue or database and a cursor that records
your position. Real tenant-change activities must be safe to repeat.
Remove the three-batch limit, but keep `WhenAll` and the history reset.
Carry only the state needed for the next batch, and keep external events that
have not yet been processed. Never continue as new while children are still running.

## Code map

Read [workflow.go](workflow.go): read a batch, start children, await all, and
continue as new. [activities.go](activities.go) reads sample batches through a
cursor and simulates the changes. [client.go](client.go) runs one coordinator;
[worker.go](worker.go) registers tasks; [main.go](main.go) starts the CLI.

## Tests

```bash
go test .
```

Tests check that the cursor gives repeatable results and respects batch limits.
They also check tenant changes, empty input, invalid saved state, and errors in
the history checks. These tests do not connect to a scheduler.
Enable the full backend tests with:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) adds a test-only observer around the
same coordinator. The SDK saves completion or continuation after the root
workflow returns. The observer can therefore pause after a batch finishes
without changing the production workflow. Each test pause has a 15-second timeout.

The test checks three different execution IDs. Each execution must have one
batch activity, five correct child results, and the expected input for the next
batch. An event sent before the first reset must survive both resets and be
processed at the end. Missing APIs or events, or reused execution IDs, fail the
test. Once enabled, the test does not skip these checks.
