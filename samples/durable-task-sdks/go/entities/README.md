# Durable entities (Go)

A durable counter keeps its state between operations. This demo signals three
changes (`+10`, `+5`, `-3`), calls the counter to read its value, then schedules a
reset five seconds later. The workflow uses durable time and timers rather than
sleeping inside an orchestrator.

## Run the demo

Use Go 1.25.0 or later and the shared module's pinned
`github.com/microsoft/durabletask-go v1.0.0-beta.1`. Configure an existing emulator
or Azure task hub using the [shared configuration guide](../README.md).
No additional Azure resources are required.

From this directory:

```bash
go run .
```

From the Go module root, use `go run ./entities`. Both forms accept
`-timeout 3m`; the default deadline is two minutes.

Expected output:

```text
Counter before scheduled reset: 12
Counter after scheduled reset: 0
```

The client waits for its workflow and prints its result. Every run uses fresh
instance and entity IDs. Completed history and counter state remain available
for inspection; unrelated entities are not queried or deleted.

## Read the code

| Read order | File | Purpose |
| --- | --- | --- |
| 1 | [counter.go](counter.go) | Counter operations, persisted value, and last-reset timestamp. |
| 2 | [workflow.go](workflow.go) | Signals, request/reply calls, and a scheduled reset. |
| 3 | [worker.go](worker.go) | Registers the counter and workflow for automatic work-item filtering. |
| 4 | [client.go](client.go) | Starts one workflow and displays its result. |
| 5 | [main.go](main.go) | Entrypoint and shared timeout handling. |

`get` returns the current integer. `snapshot` returns the value and actual reset
execution time; `delete` removes state. A scheduled signal has no reply, so the
workflow uses a bounded durable wait for delivery. Exact values and delivery-time
assertions belong to tests, not the command-line demonstration.

## Tests

Offline counter, registration, and verification-regression tests:

```bash
go test -mod=readonly .
```

Opt-in integration test against the configured task hub:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) additionally exercises direct client
signals (`100 - 25 = 75`), checks workflow completion and `12 -> 0`, proves
`read_at < due_at <= reset_at`, and reads the persisted entity state. The test
skips unless opted in and has its own bounded backend context.
