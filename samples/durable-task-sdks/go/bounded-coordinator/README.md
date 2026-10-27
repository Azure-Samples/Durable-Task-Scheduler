# Bounded coordinator (Go)

The coordinator reads one batch, starts a child orchestration for each item,
and **waits for all children**. Then it calls `ContinueAsNew` to process the
next batch with a fresh history. It carries only the batch number, the
processed count, and a cursor.

The demo processes **three batches of five simulated tenant changes**. Child
IDs include the parent ID and item ID, so batches never reuse child instances.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./bounded-coordinator`.

## Expected output

The JSON output has the coordinator instance ID and:

```json
{
  "total_batches": 3,
  "processed": 15,
  "completed": true
}
```

## Production use

Replace the sample source with a queue or database and a real cursor. Make
activities safe to repeat. Remove the three-batch limit, but keep `WhenAll` and
the history reset. Never continue as new while children are still running.

## Code map

| File | Purpose |
|---|---|
| [workflow.go](workflow.go) | Read a batch, start children, wait, continue as new |
| [activities.go](activities.go) | Cursor-based sample source and simulated changes |
| [client.go](client.go) | Runs one coordinator |
| [worker.go](worker.go) | Registers the tasks |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) checks three separate executions,
five child results per batch, and that an event sent before the first reset is
still processed at the end.
