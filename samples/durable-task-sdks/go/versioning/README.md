# Orchestration versioning (Go)

## Description

This sample runs old and new workflow behavior on one worker. Every invocation
creates unique `go-versioning-*` instance IDs and uses sample-specific registered
task names.

| Execution version | Activities and exact results |
| --- | --- |
| `1.0.0` | `Hello, World!` |
| `2.0.0` | Hello, then `Goodbye, World!` |
| `3.0.0` | Hello, goodbye, then `Notification sent: Completed greeting workflow for World` |
| `10.0.0` | Same three steps as `3.0.0` |

The worker is version **10.0.0**, configured with the SDK's
`task.VersionMatchCurrentOrOlder`. Accepting `3.0.0` on that worker exercises
numeric version ordering (`3 < 10`), which would fail with lexicographic ordering
(`"3.0.0" > "10.0.0"`). Registration-derived filters and worker dispatch both
participate. Version acceptance does not invent missing handlers: each supported
orchestration and activity version is explicitly registered.

## Prerequisites

- Go 1.25.0 or later and the shared module's pinned
  `github.com/microsoft/durabletask-go v1.0.0-beta.1`.
- An existing DTS emulator or Azure task hub. See the
  [shared emulator/live authentication setup](../README.md).
  Only task-hub data-plane access is needed.

## Run

From this directory:

```bash
go run .
```

Worker and client run together, with a two-minute default deadline. Use
`go run . -timeout 3m` to change it. Focused offline tests:

```bash
go test -mod=readonly .
```

## Expected result

Four JSON results have the versions and messages in the table above.
`activity_versions` contains the execution's version once per activity, **not**
the worker's default version for older executions. The command verifies the
persisted orchestration version, `COMPLETED` status, all messages, and all activity
versions. Its final lines are:

```text
SDK CurrentOrOlder worker 10.0.0 accepted 1.0.0, 2.0.0, 3.0.0, and 10.0.0
SAMPLE_OK versioning
```

Incorrect dispatch or results fail the command; no Azure live run is implied.

## Version handling

- The orchestration reads `ctx.Version` to select behavior for the four
  registered numeric versions. Prerelease versions are not supported by this sample.
- `10.0.0` exercises numeric version ordering and SDK worker-version matching.
- Explicit versioned registrations and inherited activity-version assertions
  demonstrate Go SDK dispatch, not just application-level branching.
- A single bounded process hosts the worker and client. It leaves completed
  instance history for inspection.
