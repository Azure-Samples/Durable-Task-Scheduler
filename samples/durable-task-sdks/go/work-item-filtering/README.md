# Work-item filtering (Go)

## Description

The Go counterpart of [Python work-item filtering](../../python/work-item-filtering/)
runs two specialized workers against the same task hub:

- **Worker A** registers only the greeting orchestration and hello activity.
- **Worker B** registers only the math orchestration and addition activity.

The shared `sample.Start` helper enables
`client.WithAutoWorkItemFilters()` separately for each registry. There are no
wildcard handlers, shared registrations, or unfiltered workers. Both workflows
are submitted through A's **client** to demonstrate that the scheduling client
does not choose the executing worker.

## Prerequisites

- Go 1.25.0 or later and the shared module's pinned
  `github.com/microsoft/durabletask-go v1.0.0-beta.1`.
- An existing DTS emulator or Azure task hub, configured with the
  [shared emulator/live authentication instructions](../README.md).
  No additional Azure resources are required.

## Run

From this directory:

```bash
go run .
```

Both worker hosts and the bounded client run in this process. The default
deadline is two minutes (`go run . -timeout 3m` overrides it). Offline tests:

```bash
go test -mod=readonly .
```

## Expected result

Both instances must actually reach `COMPLETED`. Their activity-produced worker
labels and outputs are checked before printing:

```text
Worker A: Hello, World!
Worker B: 42
SAMPLE_OK work-item-filtering
```

Missing or misrouted work, incorrect outputs, or shutdown failures cause a
nonzero exit. Instances have unique `go-filtering-*` IDs and completed history
is left for inspection.

## Differences from Python

Python uses `use_work_item_filters()` and three terminal processes. Go uses two
independent SDK hosts with registration-derived filters in one bounded process.
The greeting and math results are unchanged; the Go activity output additionally
records its worker label so routing is asserted rather than inferred from logs.
All registered names are Go/sample-specific to avoid matching Python work.
