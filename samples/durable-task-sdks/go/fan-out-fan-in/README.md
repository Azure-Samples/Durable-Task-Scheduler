# Fan-out/fan-in (Go)

The orchestration schedules all activities **before** waiting for results.
It uses `WhenAll` to wait for every activity, even if one fails. It then reads
the results and calls another activity to combine them. Each activity squares
one number. The final result contains the count, sum, and average.

The demo processes one batch containing **1–10**. It does not add random delays
or measure performance. Tasks are scheduled in parallel, but worker capacity
controls how many can run at once. Each batch can contain up to 100 items.
Numbers must be between -1,000,000 and 1,000,000 to keep calculations within `int64`.

## Prerequisites

- Go **1.25 or newer** and Docker.
- A running Durable Task Scheduler emulator with the default task hub.
- Follow [the shared Go README](../README.md) for emulator startup and live Azure
  authentication. The shared module pins SDK `v1.0.0-beta.1`.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./fan-out-fan-in`.
The process runs the worker and client together, prints the summary,
and exits after all activity work completes. Normal execution takes a few
seconds; the shared `-timeout` flag defaults to two minutes.

## Expected output

The JSON output contains a unique instance ID and this summary:

```json
{"total_items": 10, "sum": 385, "average": 38.5}
```

Open <http://localhost:8082> to view the parallel tasks and final result.
Completed history stays available. Registered names start with `GoFanOutFanIn`.
Automatic worker filters keep this sample's work separate.

## Code map

Start with [workflow.go](workflow.go): schedule all tasks, wait for the batch,
then combine results. [activities.go](activities.go) contains the calculations and result
types. [client.go](client.go) runs one batch, [worker.go](worker.go) registers
tasks, and [main.go](main.go) starts the shared command-line helper.

## Tests

Offline unit tests:

```bash
go test .
```

Tests check result totals, JSON data, empty and duplicate batches, negative
values, invalid results, and numbers that are too large. The demo does not run
these test cases. Enable [integration_test.go](integration_test.go) to check the
ten-item result and an empty batch against DTS:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```
