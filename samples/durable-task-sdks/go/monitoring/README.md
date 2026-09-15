# Monitoring — Go

Periodically poll a job-status activity, expose progress through **custom status**,
and stop when the job completes or its durable deadline expires. All
orchestration time comes from `CurrentTimeUtc`; delays use `CreateTimer`, not
`time.Sleep`. The client prints the final result; custom status remains available
in the dashboard.

The external job API is an explicitly **simulated**, stateless fixture. The
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
a 250 ms polling interval, and a 20-second safety deadline. Timers are clamped to
the deadline so the workflow never starts another poll after expiration.

Normal execution takes a few seconds. `-timeout` supplies the outer client
deadline and defaults to two minutes.

## Expected output

The JSON result contains `final_status: "Completed"` and `checks_performed: 4`,
along with unique job/instance IDs and `monitoring_duration_milliseconds`.
Elapsed duration varies with scheduler and activity latency.

All work finishes before shutdown. If the run fails, cleanup targets only
this run's instance. Nothing is purged; inspect timers, status, and results at
<http://localhost:8082>. Names are prefixed `GoMonitoring`, with automatic worker
filters.

## Production considerations

Replace only the status activity with an external API call. The finite demo
needs no history reset. A long-running production monitor should periodically
`ContinueAsNew` with compact state: job ID, last status/check count, **original
start time and absolute deadline**. Use `task.WithKeepUnprocessedEvents()` if
events can arrive, so continuation does not discard them; do not restart the
timeout budget on each execution. See [bounded coordinator](../bounded-coordinator/)
for a runnable history-reset example.

## Code map

[workflow.go](workflow.go) contains monitoring settings, the polling loop, and
deadline calculations. [activities.go](activities.go) simulates the external
status API. [client.go](client.go) monitors one job,
[worker.go](worker.go) registers tasks, and [main.go](main.go) starts the CLI.

## Tests

```bash
go test .
```

Tests cover job-state progression, never-completing jobs, invalid inputs, and
deadline clamping without connecting to a scheduler. The opt-in
[integration suite](integration_test.go) checks exact completion and timeout
results, custom-status consistency, and durable elapsed-time bounds:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```
