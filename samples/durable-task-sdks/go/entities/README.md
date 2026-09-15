# Durable entities (Go)

A durable counter saves its value between operations. This demo sends signals
for three changes (`+10`, `+5`, `-3`), reads the counter, then schedules a reset
five seconds later. The workflow uses durable time and timers.
It does not sleep inside the orchestrator.

## Run the demo

Use Go 1.25.0 or later and the SDK version set in the shared module:
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

The client waits for the workflow and prints the result. Each run uses new
instance and entity IDs. You can view the saved history and counter state
afterward. Other entities are not queried or deleted.

## Read the code

| Read order | File | Purpose |
| --- | --- | --- |
| 1 | [counter.go](counter.go) | Counter operations, saved value, and last reset time. |
| 2 | [workflow.go](workflow.go) | Signals, request/reply calls, and a scheduled reset. |
| 3 | [worker.go](worker.go) | Registers the counter and workflow for automatic work-item filtering. |
| 4 | [client.go](client.go) | Starts one workflow and displays its result. |
| 5 | [main.go](main.go) | Starts the command-line program and sets its timeout. |

`get` returns the current integer. `snapshot` returns the value and the time of
the last reset. `delete` removes the state. A scheduled signal sends no reply,
so the workflow waits for delivery with a time limit. Detailed value and timing
checks run in tests, not in the demo.

## Tests

Run counter, registration, and result-checking tests without a backend:

```bash
go test -mod=readonly .
```

Enable the integration test against your configured task hub:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) also sends signals directly from the
client (`100 - 25 = 75`). It checks workflow completion, the change from `12 -> 0`,
and the saved entity state. The timestamps must satisfy
`read_at < due_at <= reset_at`. The test runs only when enabled and has its own
timeout.
