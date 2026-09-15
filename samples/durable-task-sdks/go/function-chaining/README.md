# Function chaining — Go

Three sequential activities build a greeting: **say hello → process greeting →
finalize response**. Like the Python counterpart, each activity exchanges a typed
`Greeting` containing `recipient` and `message`; the orchestration returns the
final message. `GetInput` and `Await(&greeting)` decode the JSON boundaries into Go
structs. Every activity failure is propagated, and orchestrator logging is
replay-safe.

## Prerequisites

- Go **1.25 or newer**.
- A running Durable Task Scheduler emulator (Docker), using its default task hub.
- See [the shared Go setup](../README.md) for emulator startup, authentication,
  and live Azure configuration. This sample uses the shared module and
  `github.com/microsoft/durabletask-go v1.0.0-beta.1`.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./function-chaining`.
One process starts both the worker and client, runs one bounded greeting instead
of Python's repeated scheduling loop, verifies the exact message, and shuts down.
The default endpoint is `http://localhost:8080`; `-timeout` defaults to two minutes.
Normal execution takes a few seconds.

## Expected output

```text
{
  "instance_id": "go-function-chaining-<unique-suffix>",
  "output": "Hello User! How are you today? I hope you're doing well!"
}
SAMPLE_OK function-chaining
```

Inspect the three activity inputs and outputs at <http://localhost:8082>. History
is retained; nothing is purged. Task names are scoped with `GoFunctionChaining`,
and worker filters prevent this worker from taking other samples' tasks.

## Unit tests

```bash
go test -mod=readonly .
```

Tests cover typed payload round trips, exact transformations, malformed input,
and registration names. They do not require or substitute for a scheduler run.
