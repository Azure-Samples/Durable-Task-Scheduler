# Orchestration management (Go)

Use the DTS client to manage orchestration instances. The demo runs one batch,
restarts it with a new instance ID and the same input, and then purges (deletes)
only the two instances that this run created.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup. Your
identity needs permission to manage instances.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./orchestration-management`.

## Expected output

The service creates the restart ID.

```text
Completed batch-1: 10 items (success)
Restarted as <new-instance-id>: 10 items (success)
Service reports 2 deleted instances; only this run's IDs were submitted
```

## Code map

| File | Purpose |
| --- | --- |
| [client.go](client.go) | Start, wait, restart, and purge specific IDs |
| [workflow.go](workflow.go) | Batch orchestration with an optional release event |
| [activities.go](activities.go) | Validates and processes a batch |
| [worker.go](worker.go) | Registers tasks and starts the worker |
| [cleanup.go](cleanup.go) | Stops this run's unfinished instances after an error |
| [main.go](main.go) | Command-line entry point |

Purge uses exact IDs and does not include child instances.

## Tests

```bash
go test -mod=readonly .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) checks restart (same ID and new ID),
pause, resume, terminate, queries by time and status, and purge. After purge,
each lookup must return `api.ErrInstanceNotFound`.
