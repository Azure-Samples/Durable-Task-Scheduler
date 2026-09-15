# Orchestration versioning (Go)

Update a workflow while keeping the behavior needed by older executions.
This demo runs versions `1.0.0` and `3.0.0` on one worker.
The first says hello. The newer version also says goodbye and sends a simulated
notification.

## Run the demo

Use Go 1.25.0 or later and the SDK version set in the shared module:
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
`go-versioning-*` IDs and keeps completed history for later viewing.

## Read the code

| Read order | File | Purpose |
| --- | --- | --- |
| 1 | [workflow.go](workflow.go) | Selects activities using `ctx.Version`. |
| 2 | [activities.go](activities.go) | Produces messages and includes each activity's version. |
| 3 | [worker.go](worker.go) | Registers supported versions and configures SDK version matching. |
| 4 | [client.go](client.go) | Runs one older and one newer workflow. |
| 5 | [main.go](main.go) | Starts the command-line program and sets its timeout. |

The worker uses version `10.0.0` and `task.VersionMatchCurrentOrOlder`.
The SDK compares version numbers, so it correctly treats `3.0.0` as older than
`10.0.0`. Each supported version must be registered. Activities use the
orchestration's execution version, not the worker's default version.

The registered behaviors are hello for `1.0.0`, hello/goodbye for `2.0.0`, and all
three activities for `3.0.0` and `10.0.0`. This sample does not support prerelease
version strings.

## Tests

Run workflow decision, activity, registration, and result-checking tests offline:

```bash
go test -mod=readonly .
```

Enable the integration test against your configured task hub:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) runs all four versions and checks
completion, saved execution versions, exact messages, and each activity version.
It checks SDK version matching on real work, not only decisions in local code.
The test runs only when enabled.
