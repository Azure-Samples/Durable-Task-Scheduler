# Work item filtering (Go)

Two workers share one task hub, and each registers only its own tasks. Worker A
runs the greeting orchestration and activity. Worker B runs the math
orchestration and activity. `client.WithAutoWorkItemFilters()` sends each task
to the worker that registered it. The client that starts the orchestration
does not choose the worker.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./work-item-filtering`.

## Expected output

```text
Worker A: Hello, World!
Worker B: 42
```

## Code map

| File | Purpose |
|---|---|
| [worker.go](worker.go) | Separate task registries for each worker |
| [workflow.go](workflow.go) | Greeting and math orchestrations |
| [activities.go](activities.go) | Activities that return a result with a worker label |
| [client.go](client.go) | Starts both workers and orchestrations |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test -mod=readonly .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) starts both workers and checks the
worker labels and results.
