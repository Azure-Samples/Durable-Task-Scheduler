# Function chaining (Go)

Three activities run in order to build a greeting: **say hello → process
greeting → finalize response**. Each activity passes a `Greeting` value to the
next one. The orchestration returns the final message.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./function-chaining`.

## Expected output

```text
{
  "instance_id": "go-function-chaining-<unique-suffix>",
  "output": "Hello User! How are you today? I hope you're doing well!"
}
```

View the activity inputs and outputs at <http://localhost:8082>.

## Code map

| File | Purpose |
|---|---|
| [workflow.go](workflow.go) | The three-step orchestration |
| [activities.go](activities.go) | How each step changes the greeting |
| [client.go](client.go) | Starts one instance and prints the result |
| [worker.go](worker.go) | Registers the tasks |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

Unit tests run offline. [integration_test.go](integration_test.go) checks the
completed greeting against a real backend.
