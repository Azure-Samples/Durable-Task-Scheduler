# Orchestration versioning (Go)

Evolve a workflow without changing the behavior selected by older executions.
This demo runs versions `1.0.0` and `3.0.0` on the same worker: the first says hello;
the newer version also says goodbye and sends a simulated notification.

## Run the demo

Use Go 1.25.0 or later with the shared module's pinned
`github.com/microsoft/durabletask-go v1.0.0-beta.1`. Configure an existing emulator
or Azure task hub using the [shared configuration guide](../README.md).

From this directory:

```bash
go run .
```

From the Go module root, use `go run ./versioning`. Both forms accept
`-timeout 3m`; the default deadline is two minutes.

Expected output:

```text
Version 1.0.0: Hello, World!
Version 3.0.0: Hello, World! | Goodbye, World! | Notification sent: Completed greeting workflow for World
```

The command waits for each execution and prints its messages. It uses unique
`go-versioning-*` IDs and leaves completed history for inspection.

## Read the code

| Read order | File | Purpose |
| --- | --- | --- |
| 1 | [workflow.go](workflow.go) | Selects activities using `ctx.Version`. |
| 2 | [activities.go](activities.go) | Produces messages and carries activity-version metadata. |
| 3 | [worker.go](worker.go) | Registers supported versions and configures SDK version matching. |
| 4 | [client.go](client.go) | Runs one older and one newer workflow. |
| 5 | [main.go](main.go) | Entrypoint and shared timeout handling. |

The worker's current version is `10.0.0`, with
`task.VersionMatchCurrentOrOlder`. The SDK compares numeric versions, so `3.0.0`
is older than `10.0.0` even though lexicographic string comparison says otherwise.
Supported versions still need explicit registrations. Activities inherit the
execution version, not the worker's default.

The registered behaviors are hello for `1.0.0`, hello/goodbye for `2.0.0`, and all
three activities for `3.0.0` and `10.0.0`. This sample does not support prerelease
version strings.

## Tests

Offline branch, activity, registration, and assertion-regression tests:

```bash
go test -mod=readonly .
```

Opt-in integration test against the configured task hub:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) runs all four versions and checks
completion, persisted execution versions, exact messages, and every inherited
activity version. It exercises SDK numeric version matching on real work rather
than only testing application branches. The test skips unless opted in.
