# Human interaction (Go)

A vacation request waits for an approval event or a **durable timer**, whichever
comes first. Approval or rejection runs a processing activity. If no answer
arrives in time, the result is `Timeout`.

The demo client approves **one request** automatically, so no input is needed.
Notifications and database updates are simulations. The rejection and timeout
cases run in the integration test.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./human-interaction`.

The orchestration waits up to ten seconds for a response. In production, send
the approval event from an API that checks the user's identity.

## Expected output

```json
{
  "request_id": "go-human-interaction-<unique-suffix>",
  "status": "Approved",
  "approver": "Console Approver"
}
```

## Code map

| File | Purpose |
|---|---|
| [workflow.go](workflow.go) | Event/timer race and timer cancellation |
| [activities.go](activities.go) | Approval data and simulated effects |
| [client.go](client.go) | Schedules the request and sends the approval |
| [worker.go](worker.go) | Registers the tasks |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) checks approval, rejection, and a
one-second timeout.
