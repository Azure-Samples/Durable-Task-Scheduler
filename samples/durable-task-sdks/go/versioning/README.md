# Orchestration versioning (Go)

Change an orchestration while older executions keep their behavior. The demo
runs versions `1.0.0` and `3.0.0` on one worker. Version 1 says hello. Version 3
also says goodbye and sends a simulated notification.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./versioning`.

## Expected output

```text
Version 1.0.0: Hello, World!
Version 3.0.0: Hello, World! | Goodbye, World! | Notification sent: Completed greeting workflow for World
```

## How it works

The worker uses version `10.0.0` with `task.VersionMatchCurrentOrOlder`. The SDK
compares version numbers, so `3.0.0` is older than `10.0.0`. Registered
behaviors:

- `1.0.0`: hello
- `2.0.0`: hello and goodbye
- `3.0.0` and `10.0.0`: all three activities

Prerelease version strings are not supported.

## Code map

| File | Purpose |
|---|---|
| [workflow.go](workflow.go) | Chooses activities with `ctx.Version` |
| [activities.go](activities.go) | Messages that include the activity version |
| [worker.go](worker.go) | Registers versions and sets version matching |
| [client.go](client.go) | Runs an older and a newer orchestration |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test -mod=readonly .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) runs all four versions and checks
the saved versions and messages.
