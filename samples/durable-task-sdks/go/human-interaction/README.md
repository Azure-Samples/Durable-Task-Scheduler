# Human interaction — Go

A vacation approval workflow submits a request, publishes `Pending` custom
status, and races an external approval event against a **durable timer**.
The winner is determined by durable history, not a Go channel or wall clock.
An approval/rejection calls the processing activity; a timeout returns `Timeout`
without manufacturing a human decision.

As in the Python sample, notification and database updates are **simulations**.
There is no email sender, approval website, or real database. Unlike Python's
interactive console, this bounded client automatically exercises **approve,
reject, and no-response timeout** and checks every exact outcome.

## Prerequisites

- Go **1.25 or newer**, Docker, and a running Durable Task Scheduler emulator.
- See [shared setup and live Azure authentication](../README.md).
- This directory uses the parent Go module and SDK `v1.0.0-beta.1`.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./human-interaction`.
Worker and client run in the same process. The client waits for each submission
to become `Pending` before raising an event. Approval/rejection windows are ten
seconds; the unattended case expires after one second. Normal execution takes a
few seconds. The outer `-timeout` defaults to two minutes.

## Expected output

Three JSON results contain unique request IDs and:

| Scenario | Status | Approver |
|---|---|---|
| approve | `Approved` | `Console Approver` |
| reject | `Rejected` | `Console Approver` |
| timeout | `Timeout` | absent |

```text
SAMPLE_OK human-interaction
```

The losing timer/event wait is cancelled and awaited; unexpected task failures
are not treated as a timeout or rejection. In production, the event would come
from an authenticated approval endpoint and the response window can be hours
(up to 24 hours with this sample's validation). Activities must make external
effects idempotent because delivery can be retried.

Inspect all three instances at <http://localhost:8082>; history is not purged.
Stable task/event names start with `GoHumanInteraction`, and automatic worker
filters isolate this sample. Error cleanup targets only its own instance.

## Unit tests

```bash
go test -mod=readonly .
```

Tests check explicit approve/reject decisions, timeout output, typed activity
payloads, missing fields, and timeout bounds. The runnable client verifies the
actual durable race against the configured scheduler.
