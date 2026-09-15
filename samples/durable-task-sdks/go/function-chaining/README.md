# Function chaining (Go)

Three activities run in order to build a greeting: **say hello → process greeting →
finalize response**. They pass a `Greeting` value with `recipient` and `message`
fields. The orchestration returns the final message.

`GetInput` and `Await(&greeting)` read JSON data into Go structs. If an activity
fails, the workflow returns the error. Its logger avoids duplicate messages
when the SDK replays saved work.

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
One process starts the worker and client, builds one greeting, prints the result,
and shuts down. Detailed checks run in the tests, not in the demo.
The default endpoint is `http://localhost:8080`; `-timeout` defaults to two minutes.
Normal execution takes a few seconds.

## Expected output

```text
{
  "instance_id": "go-function-chaining-<unique-suffix>",
  "output": "Hello User! How are you today? I hope you're doing well!"
}
```

View the three activity inputs and outputs at <http://localhost:8082>. The sample
keeps its history. Task names start with `GoFunctionChaining`. Worker filters
prevent this worker from taking tasks from other samples.

## Code map

Read [workflow.go](workflow.go) to see the three steps, then
[activities.go](activities.go) to see how each step changes the greeting.
[client.go](client.go) starts one instance and prints its result;
[worker.go](worker.go) registers the stable task names;
[main.go](main.go) starts the command-line program.

## Tests

Offline unit tests:

```bash
go test .
```

Tests check how data is sent and read, the greeting changes, invalid input,
and registered task names. Integration tests run only when you enable them.
Use the emulator or Azure backend from [shared setup](../README.md):

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) checks the exact completed greeting.
