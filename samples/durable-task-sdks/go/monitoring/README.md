# Monitoring — Go

Periodically poll a job-status activity, expose progress through **custom status**,
and stop when the job completes or its durable deadline expires. All
orchestration time comes from `CurrentTimeUtc`; delays use `CreateTimer`, not
`time.Sleep`. The client prints changed custom status and checks the terminal
output against it.

The external job API is an explicitly **simulated**, stateless fixture. The
completion case finishes on check **four**, matching the Python worker's actual
`check_count >= 3` behavior. Random timing is not used.

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
The process hosts worker and client and runs two bounded cases:

1. Four checks, 250 ms polling interval, a 20-second safety deadline.
2. A job that never completes, a two-second polling interval, and a one-second
   deadline. Its timer is clamped to the deadline, so it performs **exactly one**
   check rather than polling again after expiration.

Normal execution takes a few seconds. `-timeout` supplies the outer client
deadline and defaults to two minutes.

## Expected output

JSON status updates and final results include unique job and instance IDs:

| Case | `final_status` | `checks_performed` |
|---|---|---|
| completing job | `Completed` | `4` |
| unattended job | `Timeout` | `1` |

Durable timestamps and measured duration vary; business results and check counts
are asserted exactly. Timeout duration must be at least one second.

```text
SAMPLE_OK monitoring
```

All work finishes before shutdown. If verification fails, cleanup targets only
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
for a runnable, verified history-reset example.

## Unit tests

```bash
go test -mod=readonly .
```

Tests cover job-state progression, never-completing jobs, invalid inputs, and
deadline clamping. They do not connect to a scheduler.
