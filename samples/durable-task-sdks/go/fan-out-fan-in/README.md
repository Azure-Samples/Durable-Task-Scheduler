# Fan-out/fan-in — Go

The orchestration schedules all work-item activities **before** waiting, uses
`WhenAll` to drain the complete batch (including failed siblings), decodes each
typed result, and calls a separate aggregation activity. As in Python, each item
is squared and the final result contains its count, sum, and average.

The fixture processes **1–10**, then an **empty batch**. There are no random
sleeps: these are bounded arithmetic operations, not a concurrency benchmark.
Concurrency is visible in the scheduled tasks; actual execution concurrency
depends on worker capacity. The sample caps batches at 100 items and magnitudes
at 1,000,000 to keep arithmetic within `int64`.

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
The process runs the worker and client together, asserts both exact summaries,
and exits after all activity work completes. Normal execution takes a few
seconds; the shared `-timeout` flag defaults to two minutes.

## Expected output

Two JSON results include unique instance IDs and these summaries:

```json
{"total_items": 10, "sum": 385, "average": 38.5}
{"total_items": 0, "sum": 0, "average": 0}
```

The final line is:

```text
SAMPLE_OK fan-out-fan-in
```

Open <http://localhost:8082> to inspect the parallel activity scheduling and final
aggregation. Completed history is retained. All registered names start with
`GoFanOutFanIn`; automatic worker filters isolate this sample.

## Unit tests

```bash
go test -mod=readonly .
```

Tests cover exact aggregation, typed JSON activity boundaries, empty and
duplicate batches, negative values, invalid results, and overflow prevention.
Scheduler execution is verified separately by running the sample.
