# Sub-orchestrations — Go

A parent loads orders in an activity, fans out **child orchestrations**, waits
for every child, and aggregates the results. Each child follows the Python
domain pipeline:

**inventory → payment → shipping → customer notification**

These business operations are explicit **simulations** with no external effects.
Instead of random outcomes, five deterministic orders exercise success and each
early-exit failure path. A failed business decision returns an order-level
`failed` result; an actual activity/SDK error fails the workflow and is not
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
The process starts worker and client, schedules all five children before waiting,
asserts the full ordered result including attempted steps, then shuts down.
Normal execution takes a few seconds. The outer `-timeout` defaults to two
minutes.

## Expected output

| Order | Result | Reason | Attempted steps |
|---|---|---|---|
| order-1 | completed | — | all four |
| order-2 | failed | out of stock | inventory |
| order-3 | failed | payment failed | inventory, payment |
| order-4 | failed | shipping failed | inventory, payment, shipping |
| order-5 | failed | customer notification failed | all four |

JSON output includes **`total_completed: 1`** and **`total_failed: 4`**, followed by:

```text
SAMPLE_OK sub-orchestrations
```

The `results` array contains the detail once, rather than duplicating Python's
identical `details` array. No compensation is implied by a failed order; see the
[saga sample](../saga/) for reversing completed external operations.

Child IDs are derived deterministically from the unique parent ID and order ID.
The parent drains all children even when one fails; error cleanup can recursively
terminate only this run's own family. Completed instances remain inspectable at
<http://localhost:8082>. All task names start with `GoSubOrchestrations`, and
automatic worker filters isolate this sample.

## Unit tests

```bash
go test -mod=readonly .
```

Tests run the child decision logic through typed activity payloads and verify
exact call order, all early exits, error propagation, and fixture validity.
They do not connect to a scheduler.
