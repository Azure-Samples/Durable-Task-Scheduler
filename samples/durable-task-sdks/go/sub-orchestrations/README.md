# Sub-orchestrations (Go)

A parent orchestration loads orders and starts one **child orchestration** per
order in parallel. It waits for all children and combines the results. Each
child runs **inventory → payment → shipping → customer notification**. All
steps are simulations.

A business rejection marks that order as `failed`. An activity or SDK error
fails the whole orchestration.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./sub-orchestrations`.

## Expected output

| Order | Result | Steps |
|---|---|---|
| order-1 | completed | all four |
| order-2 | completed | all four |

The JSON output has **`total_completed: 2`**, **`total_failed: 0`**, the parent
instance ID, and each child's result. A failed order does not undo completed
steps; see the [saga sample](../saga/) for that.

## Code map

| File | Purpose |
|---|---|
| [workflow.go](workflow.go) | Parent fan-out and child steps |
| [activities.go](activities.go) | Simulated order source and operations |
| [client.go](client.go) | Runs one parent and prints the summary |
| [worker.go](worker.go) | Registers the tasks |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) runs five orders and expects one
completed and four failed.
