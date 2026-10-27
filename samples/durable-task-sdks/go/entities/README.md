# Durable entities (Go)

A durable counter keeps its value between operations. The demo sends three
signals (`+10`, `+5`, `-3`), reads the counter, and schedules a reset five
seconds later.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./entities`.

## Expected output

```text
Counter before scheduled reset: 12
Counter after scheduled reset: 0
```

Each run uses new instance and entity IDs.

## Code map

| File | Purpose |
|---|---|
| [counter.go](counter.go) | Counter operations (`add`, `subtract`, `reset`, `get`, `snapshot`, `delete`) and state |
| [workflow.go](workflow.go) | Signals, request/reply calls, and the scheduled reset |
| [worker.go](worker.go) | Registers the entity and orchestration |
| [client.go](client.go) | Starts one orchestration and prints the result |
| [main.go](main.go) | Command-line entry point |

A scheduled signal sends no reply, so the orchestration waits for delivery with
a time limit.

## Tests

```bash
go test -mod=readonly .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) also signals the entity from the
client (`100 - 25 = 75`). It checks the reset from `12` to `0` and that
`read_at < due_at <= reset_at`.
