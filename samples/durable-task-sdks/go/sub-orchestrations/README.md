# Sub-orchestrations — Go

A parent loads orders in an activity, fans out **child orchestrations**, waits
for every child, and aggregates the results. Each child follows this
order-processing pipeline:

**inventory → payment → shipping → customer notification**

These business operations are explicit **simulations** with no external effects.
The demo fulfills two deterministic orders. A failed business decision returns
an order-level `failed` result; an actual activity/SDK error fails the workflow and is not
converted to an expected business rejection.

## Prerequisites

- Go **1.25 or newer**, Docker, and a running Durable Task Scheduler emulator.
- Follow [shared setup and live Azure authentication](../README.md).
- Uses the parent Go module and SDK `v1.0.0-beta.1`.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./sub-orchestrations`.
The process starts worker and client, schedules both children before waiting,
prints their ordered results, then shuts down.
Normal execution takes a few seconds. The outer `-timeout` defaults to two
minutes.

## Expected output

| Order | Result | Reason | Attempted steps |
|---|---|---|---|
| order-1 | completed | — | all four |
| order-2 | completed | — | all four |

JSON output includes **`total_completed: 2`** and **`total_failed: 0`**, plus the
parent instance ID and detailed child results.

The `results` array contains each order's outcome and completed steps.
No compensation is implied by a failed order; see the
[saga sample](../saga/) for reversing completed external operations.

Child IDs are derived deterministically from the unique parent ID and order ID.
The parent drains all children even when one fails; error cleanup can recursively
terminate only this run's own family. Completed instances remain inspectable at
<http://localhost:8082>. All task names start with `GoSubOrchestrations`, and
automatic worker filters isolate this sample.

## Code map

Read [workflow.go](workflow.go) for parent fan-out and each child's ordered steps,
then [activities.go](activities.go) for the simulated order source and operations.
[client.go](client.go) runs one parent and prints its summary;
[worker.go](worker.go) registers handlers; [main.go](main.go) starts the CLI.

## Tests

```bash
go test .
```

Tests run the child decision logic through typed activity payloads and verify
exact call order, all early exits, error propagation, and fixture validity.
They do not connect to a scheduler. The opt-in
[integration suite](integration_test.go) supplies a five-order source fixture
to the same parent/child workflows and business activities. It verifies success,
each early failure, exact attempted steps, and the one-completed/four-failed
aggregate. The exhaustive fixture is not part of the runnable demo.

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```
