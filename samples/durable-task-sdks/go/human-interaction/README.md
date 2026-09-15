# Human interaction (Go)

A vacation approval workflow submits a request and sets its custom status to
`Pending`. It waits for either an approval event or a **durable timer**.
Saved workflow history determines which arrives first.
An approval or rejection starts the processing activity. If no response arrives
in time, the workflow returns `Timeout` without assuming a decision.

Notification and database updates are **simulations**.
There is no email sender, approval website, or real database. The demo client
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
after scheduling the request. DTS stores the event if the workflow is not waiting
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

The workflow cancels the other wait and waits for that cancellation to finish.
Unexpected errors are not treated as a timeout or rejection. In a production
app, the approval event should come from an API that checks the user's identity.
This sample allows a response period of up to 24 hours.
Activities may run more than once, so repeating a call must not repeat its
external effects.

View the instance at <http://localhost:8082>. Its history is not deleted.
Stable task/event names start with `GoHumanInteraction`, and automatic worker
filters keep this sample's work separate. If an error occurs, cleanup affects
only the instance created by this run.

## Code map

Read [workflow.go](workflow.go) for the event/timer race and cancellation,
then [activities.go](activities.go) for approval payloads and simulated effects.
[client.go](client.go) schedules the request and supplies the approval;
[worker.go](worker.go) registers the handlers;
[main.go](main.go) starts the command-line program.

## Tests

Offline tests:

```bash
go test .
```

Tests check approval and rejection decisions, timeout output, activity data,
missing fields, and timeout limits. When enabled, the
[integration suite](integration_test.go) waits for `Pending` status and verifies
approval, rejection, and one-second timeout results when no response is sent:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```
