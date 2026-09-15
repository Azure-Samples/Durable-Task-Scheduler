# Scheduled tasks (Go)

Use the Go SDK's schedule helpers to start report workflows at regular intervals.
The demo starts a report every five seconds, then pauses the schedule.
It changes the interval and region, resumes the schedule, and deletes it.
No external cron service is needed.

## Run the demo

Use Go 1.25.0 or later and the SDK version set in the shared module:
`github.com/microsoft/durabletask-go v1.0.0-beta.1`. Configure an existing emulator
or Azure task hub using the [shared configuration guide](../README.md).
No additional Azure resources are required.

Use **schedule state managed by Go workers**. Do not let workers from other SDKs
manage the same schedule entities. Shared names do not make their state formats
compatible. SDK system handlers have fixed names. This sample uses its own
application handler names and schedule IDs.

From this directory:

```bash
go run .
```

From the Go module root, use `go run ./scheduled-tasks`. Both forms accept
`-timeout 3m`; the default deadline is two minutes.

Example output (IDs, report counts, and line order may vary):

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

The activity prints each report it generates. The demo waits for eleven seconds
at first and five seconds after resuming. This waiting time does not prove how
many workflows completed. The integration test checks the actual executions.

## Read the code

| Read order | File | Purpose |
| --- | --- | --- |
| 1 | [client.go](client.go) | Creates and manages one short-interval schedule. |
| 2 | [workflow.go](workflow.go) | Report workflow and its input/output types. |
| 3 | [activities.go](activities.go) | Generates and prints a report. |
| 4 | [worker.go](worker.go) | Registers application and SDK system handlers. |
| 5 | [cleanup.go](cleanup.go) | Deletes this run's schedule, even if creation times out. |
| 6 | [main.go](main.go) | Starts the command-line program and sets its timeout. |

`RegisterScheduledTasks` registers the `Schedule` entity and two system
orchestrators. `WithScheduledTasks` tells DTS that the worker supports schedules.
These system orchestrators have no version, and automatic filters include all
registered tasks. This beta has no public run-now method. Reports start through
the recurring schedule, not through separate manual requests.

The client keeps the schedule handle before it asks DTS to create the schedule.
If the result is uncertain, cleanup waits for the creation result before deleting.
Cleanup has a separate 30-second timeout, and the worker stays running during
that period. Cancelling the original context can still stop connection setup.
The schedule also has a 90-second `EndAt` limit, but a worker must process that
limit. Cleanup errors are returned to the caller.

Deletion stops future scheduled starts. It does not stop report workflows that
have already started. Completed report and SDK operation history remain
available. The demo and tests never delete unrelated schedules or clear a whole hub.

## Tests

Run registration, data, cleanup-order, and result-checking tests offline:

```bash
go test -mod=readonly .
```

Enable the integration test against your configured task hub:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

[integration_test.go](integration_test.go) checks schedule creation, lookup, and
listing. It requires two different completed reports before the update.
While paused, the schedule must not advance or start updated reports during two
intervals. After resume, the saved interval and input must produce updated output.
After deletion, the schedule must no longer exist.

Queries use only this run's schedule and target ID prefixes. Duplicate or
unfinished instances do not count as completed runs. A successful delete
response is not enough without an absence check.
The test runs only when enabled and has its own timeout.
