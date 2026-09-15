# Scheduled tasks (Go)

Use the Go SDK's recurring schedule helpers to start report workflows
periodically. The demo creates a five-second schedule, lets reports print, pauses
it, updates the interval and region, resumes it, and deletes its schedule.
No external cron service is involved.

## Run the demo

Use Go 1.25.0 or later with the shared module's pinned
`github.com/microsoft/durabletask-go v1.0.0-beta.1`. Configure an existing emulator
or Azure task hub using the [shared configuration guide](../README.md).
No additional Azure resources are required.

Use **Go-owned schedule state**. Do not mix schedule-worker implementations
against the same entities or assume cross-SDK schedule interoperability.
SDK system handlers have fixed names; application handlers and schedule IDs
are sample-specific.

From this directory:

```bash
go run .
```

From the Go module root, use `go run ./scheduled-tasks`. Both forms accept
`-timeout 3m`; the default deadline is two minutes.

Representative output (IDs, report counts, and interleaving vary):

```text
Schedule go-scheduled-tasks-<unique>: westus reports every 5s
Report for 'westus' generated
Report for 'westus' generated
Paused schedule
Resumed schedule: eastus reports every 2s
Report for 'eastus' generated
Report for 'eastus' generated
Deleted schedule go-scheduled-tasks-<unique>
```

The activity prints each report it actually generates. The command observes
reports for eleven seconds initially and five seconds after resuming; it does
not interpret elapsed time as proof of how many workflows completed. Exact
execution checks are in the opt-in integration test.

## Read the code

| Read order | File | Purpose |
| --- | --- | --- |
| 1 | [client.go](client.go) | Creates and manages one short-interval schedule. |
| 2 | [workflow.go](workflow.go) | Report workflow and its input/output types. |
| 3 | [activities.go](activities.go) | Generates and prints a report. |
| 4 | [worker.go](worker.go) | Registers application and SDK system handlers. |
| 5 | [cleanup.go](cleanup.go) | Safely deletes the owned schedule, including after creation timeouts. |
| 6 | [main.go](main.go) | Entrypoint and shared timeout handling. |

`RegisterScheduledTasks` installs the `Schedule` entity and the two system
orchestrators. `WithScheduledTasks` advertises the capability and keeps system
orchestrators unversioned; automatic work-item filters include all registrations.
There is no public run-now method in this beta: targets start through recurring
ticks, not substitute manual scheduling.

Cleanup retains the schedule handle before creation, waits for an uncertain
creation outcome before deleting, and uses a fresh 30-second context while the
worker remains alive. Connection setup still honors the original cancellation
context. A finite 90-second `EndAt` is a secondary safeguard whose processing
also requires a worker. Cleanup errors are returned.

Deletion stops future ticks, not already-started finite report workflows.
Completed report and SDK operation history remain for inspection. Neither the
demo nor its tests perform broad purges or delete unrelated schedules.

## Tests

Offline registration, payload, cleanup-ordering, and verification-regression tests:

```bash
go test -mod=readonly .
```

Opt-in integration test against the configured task hub:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) verifies create/read/list; two distinct
completed initial reports; paused status with no schedule advancement or updated
targets during two intervals; persisted interval/input updates; active status
and updated output after resume; and actual absence after delete.
Queries are restricted to this run's schedule/target prefixes. Duplicate or
unfinished instances never count as completed runs, and a successful delete
response alone cannot satisfy deletion verification. The test skips unless
opted in and uses a bounded real-backend context.
