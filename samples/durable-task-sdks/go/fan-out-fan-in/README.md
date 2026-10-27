# Fan-out/fan-in (Go)

The orchestration schedules one activity per number, then uses `WhenAll` to
wait for all of them. Each activity squares a number. A final activity combines
the results into a count, sum, and average.

The demo processes the numbers **1–10**. A batch can have up to 100 items.
Each number must be between -1,000,000 and 1,000,000.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./fan-out-fan-in`.

## Expected output

The JSON output has a unique instance ID and this summary:

```json
{"total_items": 10, "sum": 385, "average": 38.5}
```

## Code map

| File | Purpose |
|---|---|
| [workflow.go](workflow.go) | Schedules all tasks, waits, and combines results |
| [activities.go](activities.go) | Calculations and result types |
| [client.go](client.go) | Runs one batch |
| [worker.go](worker.go) | Registers the tasks |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

Unit tests check totals, invalid input, and overflow. The
[integration test](integration_test.go) checks the ten-item batch and an empty
batch against DTS.
