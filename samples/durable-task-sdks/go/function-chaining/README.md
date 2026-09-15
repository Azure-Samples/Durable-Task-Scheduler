# Function chaining — Go

Three sequential activities build a greeting: **say hello → process greeting →
finalize response**. Each activity exchanges a typed
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
One process starts both the worker and client, runs one bounded greeting,
prints the result, and shuts down. Exhaustive verification belongs to the tests,
not the runnable demo.
The default endpoint is `http://localhost:8080`; `-timeout` defaults to two minutes.
Normal execution takes a few seconds.

## Expected output

```text
{
  "instance_id": "go-function-chaining-<unique-suffix>",
  "output": "Hello User! How are you today? I hope you're doing well!"
}
```

Inspect the three activity inputs and outputs at <http://localhost:8082>. History
is retained; nothing is purged. Task names are scoped with `GoFunctionChaining`,
and worker filters prevent this worker from taking other samples' tasks.

## Code map

Read [workflow.go](workflow.go) for the three awaited steps, then
[activities.go](activities.go) for the typed greeting transformations.
[client.go](client.go) starts one instance and prints its result;
[worker.go](worker.go) registers the stable task names;
[main.go](main.go) is only the CLI entrypoint.

## Tests

Offline unit tests:

```bash
go test .
```

Tests cover typed payload round trips, exact transformations, malformed input,
and registration names. Integration tests are skipped unless explicitly enabled.
Against the emulator or live backend configured through [shared setup](../README.md):

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) checks the exact completed greeting.
