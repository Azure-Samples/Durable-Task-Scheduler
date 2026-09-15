# Sub-orchestrations (Go)

A parent workflow loads orders through an activity and starts **child
orchestrations** in parallel. It waits for every child and combines the results.
Each child follows these order-processing steps:

**inventory → payment → shipping → customer notification**

These operations are **simulations** and do not affect external services.
The demo completes two orders using fixed sample data. A rejected business
request returns `failed` for that order. An activity or SDK error fails the
workflow and is not treated as a normal business rejection.

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
| order-1 | completed | None | all four |
| order-2 | completed | None | all four |

JSON output includes **`total_completed: 2`** and **`total_failed: 0`**, plus the
parent instance ID and detailed child results.

The `results` array contains each order's outcome and completed steps.
A failed order does not automatically undo completed steps. See the
[saga sample](../saga/) for an example that reverses completed operations.

Child IDs are built from the unique parent ID and order ID. They stay the same
when work is replayed. The parent waits for all children, even if one fails.
Error cleanup can stop only this run's parent and children. View completed instances at
<http://localhost:8082>. All task names start with `GoSubOrchestrations`, and
automatic worker filters keep this sample's work separate.

## Code map

Read [workflow.go](workflow.go) for parent fan-out and each child's ordered steps,
then [activities.go](activities.go) for the simulated order source and operations.
[client.go](client.go) runs one parent and prints its summary;
[worker.go](worker.go) registers handlers; [main.go](main.go) starts the CLI.

## Tests

```bash
go test .
```

Tests check the child's decisions, activity data, call order, early stops, and
errors. They do not connect to a scheduler. When enabled, the
[integration suite](integration_test.go) supplies five sample orders to the
same workflows and activities. It checks success and each failure case,
including which steps were attempted. The total must be one completed order
and four failed orders. These extra cases are not part of the demo.

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```
