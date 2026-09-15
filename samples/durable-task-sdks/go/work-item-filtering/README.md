# Work-item filtering (Go)

Run specialized workers in one task hub without giving every worker every
handler. Worker A knows the greeting workflow and hello activity; worker B knows
the math workflow and addition activity.

The shared `sample.Start` helper enables `client.WithAutoWorkItemFilters()` for
each independent registry. Both workflows are submitted through A's client:
the client's connection does not select which worker executes the work.

## Run the demo

Use Go 1.25.0 or later with the shared module's pinned
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

The command waits for both workflows and prints their activity-produced results.
Every invocation uses unique `go-filtering-*` IDs; completed history remains
available for inspection.

## Read the code

| Read order | File | Purpose |
| --- | --- | --- |
| 1 | [worker.go](worker.go) | Builds two disjoint registries with no wildcard handlers. |
| 2 | [workflow.go](workflow.go) | Greeting and math workflows, each calling its own activity. |
| 3 | [activities.go](activities.go) | Returns the greeting or sum with a worker label. |
| 4 | [client.go](client.go) | Hosts both workers, submits both workloads, and displays results. |
| 5 | [main.go](main.go) | Entrypoint and shared timeout handling. |

Sample-specific registered names keep these workers separate from unrelated
samples. Each worker is shut down independently on success or failure.

## Tests

Offline registry-isolation and activity tests:

```bash
go test -mod=readonly .
```

Opt-in integration test against the configured task hub:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) starts both real workers, requires
both workflows to complete, and checks exact worker labels, the greeting, and
the sum. The test skips unless opted in and uses a bounded backend context.
