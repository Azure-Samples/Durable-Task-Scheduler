# Scheduled tasks (Go)

Use the SDK's schedule helpers to start a report orchestration at a fixed
interval. The demo starts a report every five seconds, pauses the schedule,
changes the interval and region, resumes it, and deletes it.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

Use a task hub where only Go workers manage schedules. Go schedule state is not
compatible with other SDKs.

## Run

From this directory:

```bash
go run .
```

Or, from the Go samples directory: `go run ./scheduled-tasks`.

Example output (IDs, counts, and order can vary):

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

## How it works

- `RegisterScheduledTasks` registers the `Schedule` entity and two system
  orchestrations. `WithScheduledTasks` tells DTS that the worker supports
  schedules.
- This beta has no public "run now" method. Reports start only from the schedule.
- The schedule has a 90-second `EndAt` limit. Cleanup always deletes this run's
  schedule, with a 30-second limit.
- Deleting a schedule stops future runs. It does not stop reports that already
  started.

## Code map

| File | Purpose |
| --- | --- |
| [client.go](client.go) | Creates and manages one schedule |
| [workflow.go](workflow.go) | Report orchestration and its types |
| [activities.go](activities.go) | Generates and prints a report |
| [worker.go](worker.go) | Registers app and SDK system tasks |
| [cleanup.go](cleanup.go) | Deletes this run's schedule |
| [main.go](main.go) | Command-line entry point |

## Tests

```bash
go test -mod=readonly .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The [integration test](integration_test.go) checks create, get, list, pause,
update, resume, and delete. It counts the reports that actually completed.
