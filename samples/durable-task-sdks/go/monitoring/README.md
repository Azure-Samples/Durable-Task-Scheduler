# Monitoring (Go)

The orchestration checks a job's status at a fixed interval and reports progress
through **custom status**. It stops when the job finishes or the deadline
passes. It uses `CurrentTimeUtc` and `CreateTimer`, not `time.Sleep`.

The job is simulated. It reports `Running` three times and `Completed` on the
**fourth** check. The demo uses a 250 ms interval and a 20-second deadline.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./monitoring`.

## Expected output

The JSON result has `final_status: "Completed"` and `checks_performed: 4`, plus
the job ID, instance ID, and `monitoring_duration_milliseconds`.

## Production use

Replace the status activity with a call to your job API. For a long-running
monitor, call `ContinueAsNew` from time to time. Carry the job ID, last status,
check count, and the **original start time and deadline**. Use
`task.WithKeepUnprocessedEvents()`. See [bounded coordinator](../bounded-coordinator/)
for an example.

## Code map

| File | Purpose |
|---|---|
| [workflow.go](workflow.go) | Polling loop and deadline logic |
| [activities.go](activities.go) | Simulated status API |
| [client.go](client.go) | Monitors one job |
| [worker.go](worker.go) | Registers the tasks |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) checks the completion and timeout
cases and the custom status.
