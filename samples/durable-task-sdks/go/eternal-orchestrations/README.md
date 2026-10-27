# Eternal orchestrations (Go)

The orchestration runs a cleanup activity, waits on a durable timer, and then
calls **continue as new**. It keeps the same instance ID but starts with a new,
empty history. It carries only the cycle count and the total number of removed
records.

The demo stops after **five cycles** with 250 ms timers. Cleanup is an
in-memory simulation. Nothing is deleted.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./eternal-orchestrations`.

## Expected output

```json
{"iterations": 5, "total_removed": 10, "last_message": "Cleanup completed"}
```

## Production use

Replace the sample data and the five-cycle limit with your own source and stop
rule. Carry only small state between executions. The code uses
`task.WithKeepUnprocessedEvents()` so that pending events are not lost. Make
cleanup activities safe to run more than once.

## Code map

| File | Purpose |
|---|---|
| [workflow.go](workflow.go) | Cleanup, timer, and continue-as-new loop |
| [activities.go](activities.go) | Simulated cleanup |
| [client.go](client.go) | Starts one instance and prints the result |
| [worker.go](worker.go) | Registers the tasks |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) checks five cycles and ten removals.
It also checks that the latest history contains only the last cycle.
