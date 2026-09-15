# Eternal orchestrations (Go)

Run a cleanup activity at regular intervals and wait with a durable timer.
Then **continue as new**, carrying only the cycle count and total number of
removed records. The instance ID stays the same, but execution history starts
again.

The demo stops after **five cycles** and uses 250 ms durable timers.
Cleanup is an **in-memory simulation**. Each cycle finds two expired records and
keeps one current record. No user files, database rows, or scheduler instances
are deleted.

## Prerequisites

- Go **1.25 or newer**, Docker, and a running Durable Task Scheduler emulator.
- See [shared emulator and live Azure setup](../README.md).
- The parent module pins Durable Task Go SDK `v1.0.0-beta.1`.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./eternal-orchestrations`.
The worker and client run together. The client waits for all cycles and prints
the final cleanup result. It does not read history or run detailed test checks.

Normal execution takes a few seconds; the outer `-timeout` defaults to two
minutes. No recurring work remains when the process exits.

## Expected output

The JSON output includes the instance ID and this result:

```json
{"iterations": 5, "total_removed": 10, "last_message": "Cleanup completed"}
```

View the latest execution at <http://localhost:8082>. `ContinueAsNew` resets
history; it does **not** use a purge command. Registered names start with
`GoEternal`, and automatic worker filters keep this sample's work separate.

## Production continuation

For a long-running workflow, replace the sample data and five-cycle limit with
your cleanup source and a clear rule for stopping. Keep only a small amount of
state between executions. Do not carry a list of results that grows forever.
The code uses `task.WithKeepUnprocessedEvents()` to keep events that have not yet
been processed. Finish activities, timers, and child work before resetting
history. Cleanup activities may run more than once, so repeated calls must not
cause duplicate effects.

## Code map

Read [workflow.go](workflow.go) for the cleanup/timer/continuation sequence.
[activities.go](activities.go) contains the simulated cleanup and its result.
[client.go](client.go) starts one recurring instance and prints its final result;
[worker.go](worker.go) registers tasks; [main.go](main.go) starts the CLI.

## Tests

```bash
go test .
```

Tests check how sample records are grouped, the cleanup results, invalid state,
and whether history has reset. These tests do not connect to a scheduler.
When enabled, the [integration suite](integration_test.go) checks five cycles
and ten removals. The latest history must contain only cycle five, one cleanup
activity, and one completed timer. If the history API is unavailable, the test
fails rather than skipping the check.

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```
