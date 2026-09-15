# Human interaction — Go

A vacation approval workflow submits a request, publishes `Pending` custom
status, and races an external approval event against a **durable timer**.
The winner is determined by durable history, not a Go channel or wall clock.
An approval/rejection calls the processing activity; a timeout returns `Timeout`
without manufacturing a human decision.

Notification and database updates are **simulations**.
There is no email sender, approval website, or real database. The bounded client
automatically approves **one vacation request**, so the demo needs no interactive
input. The rejection and timeout scenarios belong to the integration tests.

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
Worker and client run in the same process. The client raises an approval event
after scheduling the request; DTS buffers it if the workflow is not waiting
yet. The workflow has a ten-second response window. Normal execution takes a few
seconds. The outer `-timeout` defaults to two minutes and accepts `-timeout 3m`.

## Expected output

```json
{
  "request_id": "go-human-interaction-<unique-suffix>",
  "status": "Approved",
  "approver": "Console Approver"
}
```

The losing timer/event wait is cancelled and awaited; unexpected task failures
are not treated as a timeout or rejection. In production, the event would come
from an authenticated approval endpoint and the response window can be hours
(up to 24 hours with this sample's validation). Activities must make external
effects idempotent because delivery can be retried.

Inspect the instance at <http://localhost:8082>; history is not purged.
Stable task/event names start with `GoHumanInteraction`, and automatic worker
filters isolate this sample. Error cleanup targets only its own instance.

## Code map

Read [workflow.go](workflow.go) for the event/timer race and cancellation,
then [activities.go](activities.go) for approval payloads and simulated effects.
[client.go](client.go) schedules the request and supplies the approval;
[worker.go](worker.go) registers the handlers;
[main.go](main.go) is the thin entrypoint.

## Tests

Offline tests:

```bash
go test .
```

Tests check explicit approve/reject decisions, timeout output, typed activity
payloads, missing fields, and timeout bounds. The opt-in
[integration suite](integration_test.go) waits for `Pending` status and verifies
exact approval, rejection, and one-second unattended timeout outcomes:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```
