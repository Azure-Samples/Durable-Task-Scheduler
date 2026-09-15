# Work-item filtering (Go)

Run workers for different tasks in one task hub. Each worker registers only the
handlers it needs. Worker A runs the greeting workflow and hello activity.
Worker B runs the math workflow and addition activity.

The shared `sample.Start` helper enables `client.WithAutoWorkItemFilters()` for
each worker's registry. Both workflows are started through A's client.
The client's connection does not choose which worker runs the tasks.

## Run the demo

Use Go 1.25.0 or later and the SDK version set in the shared module:
`github.com/microsoft/durabletask-go v1.0.0-beta.1`. Configure an existing emulator
or Azure task hub using the [shared configuration guide](../README.md).
No additional Azure resources are needed.

From this directory:

```bash
go run .
```

From the Go module root, use `go run ./work-item-filtering`. Both forms accept
`-timeout 3m`; the default deadline is two minutes.

Expected output:

```text
Worker A: Hello, World!
Worker B: 42
```

The command waits for both workflows and prints the activity results.
Each run uses unique `go-filtering-*` IDs. Completed history stays available
for later viewing.

## Read the code

| Read order | File | Purpose |
| --- | --- | --- |
| 1 | [worker.go](worker.go) | Builds separate task registries without catch-all handlers. |
| 2 | [workflow.go](workflow.go) | Greeting and math workflows, each calling its own activity. |
| 3 | [activities.go](activities.go) | Returns the greeting or sum with a worker label. |
| 4 | [client.go](client.go) | Starts both workers and workflows, then displays results. |
| 5 | [main.go](main.go) | Starts the command-line program and sets its timeout. |

Sample-specific registered names keep these workers separate from unrelated
samples. Each worker is shut down independently on success or failure.

## Tests

Run tests for separate registries and activity behavior without a backend:

```bash
go test -mod=readonly .
```

Enable the integration test against your configured task hub:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) starts both real workers, requires
both workflows to complete, and checks exact worker labels, the greeting, and
the sum. The test runs only when enabled and has its own timeout.
