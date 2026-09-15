# Monitoring (Go)

Check a job's status at regular intervals and show progress through **custom
status**. Stop when the job finishes or reaches its deadline.
The orchestration reads time from `CurrentTimeUtc` and uses `CreateTimer` for
delays, not `time.Sleep`. The client prints the final result. You can also view
custom status in the dashboard.

The job API uses **simulated data** and keeps no state between calls. The
completion case reports `Running` for the first three checks and `Completed` on
check **four**. Random timing is not used.

## Prerequisites

- Go **1.25 or newer**.
- Docker and a running Durable Task Scheduler emulator.
- See [shared setup and live authentication](../README.md). The shared module
  pins Durable Task Go SDK `v1.0.0-beta.1`.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./monitoring`.
The process hosts worker and client and monitors one simulated job: four checks,
a 250 ms interval, and a 20-second deadline. Timer delays cannot extend past
that deadline, so the workflow does not start another check after time runs out.

Normal execution takes a few seconds. `-timeout` limits the total client runtime
and defaults to two minutes.

## Expected output

The JSON result contains `final_status: "Completed"` and `checks_performed: 4`,
along with unique job/instance IDs and `monitoring_duration_milliseconds`.
The duration depends on how long the scheduler and activities take to respond.

All work finishes before shutdown. If the run fails, cleanup targets only
this run's instance. Nothing is deleted. View timers, status, and results at
<http://localhost:8082>. Names are prefixed `GoMonitoring`, with automatic worker
filters.

## Production considerations

Replace the status activity with a call to your job API. This short demo does
not need to reset its history. A long-running monitor should call
`ContinueAsNew` at regular points. Carry only the job ID, last status, check
count, **original start time, and original deadline**.
Use `task.WithKeepUnprocessedEvents()` to keep events that have not yet been
processed. Do not restart the timeout with each new execution.
See [bounded coordinator](../bounded-coordinator/) for a history-reset example.

## Code map

[workflow.go](workflow.go) contains monitoring settings, the polling loop, and
deadline calculations. [activities.go](activities.go) simulates the external
status API. [client.go](client.go) monitors one job,
[worker.go](worker.go) registers tasks, and [main.go](main.go) starts the CLI.

## Tests

```bash
go test .
```

Tests check status changes, jobs that never finish, invalid inputs, and timer
limits without connecting to a scheduler. When enabled, the
[integration suite](integration_test.go) checks exact completion and timeout
results, matching custom status, and elapsed-time limits:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```
