# Scheduled tasks (Go)

## Description

This sample uses the **published Go SDK schedule helpers** to create, read, list, pause,
update, resume, run, and delete a recurring report schedule.

- The initial schedule runs every **five seconds**, generating
  `Report for 'westus' generated`. At least two distinct target instances must
  actually complete with that output.
- While paused, a sparse update changes the interval to **two seconds** and the
  input region to `eastus`. The command observes two updated intervals, verifying
  the schedule does not advance and no updated target starts.
- After resuming, at least one target must complete with
  `Report for 'eastus' generated`.
- Deletion must be followed by `Describe` returning `ErrScheduleNotFound` and
  `Get` returning `nil`.

No external cron service or additional Azure resource is required.

## Prerequisites

- Go 1.25.0 or later and the shared module's pinned
  `github.com/microsoft/durabletask-go v1.0.0-beta.1`.
- An existing DTS emulator or Azure task hub. Follow the
  [shared emulator/live authentication setup](../README.md).
- Use this Go schedule implementation only with **Go-owned schedule state**.
  Do not mix schedule-worker implementations against the same entities or
  assume cross-SDK schedule interoperability. The system handlers have fixed SDK
  names; application report names and schedule/target IDs are Go/sample-specific.

## Run

From this directory:

```bash
go run .
```

The client and worker run together with a two-minute scenario deadline.
`go run . -timeout 3m` changes that deadline. Offline tests:

```bash
go test -mod=readonly .
```

## Expected result

The unique schedule ID and run counts vary. Successful verification prints:

```text
Created/read/listed schedule go-scheduled-tasks-<unique>
Verified initial recurring reports: <count> (at least 2), Report for 'westus' generated
Verified pause and sparse update: no updated runs during two intervals
Verified resumed reports: <count> (at least 1), Report for 'eastus' generated
Deleted owned schedule; Describe reports not found and Get returns nil
SAMPLE_OK scheduled-tasks
```

Counts are observed completed orchestration instances, not an estimate from
sleep duration or schedule metadata. Target inputs, outputs, and terminal statuses
are fetched and checked. Query results must match this schedule's ID prefix and
registered target name; missing/broken APIs do not produce a success marker.

## Registration and cleanup

`durabletaskscheduler.RegisterScheduledTasks(registry)` registers the SDK's
`Schedule` entity, `ExecuteScheduleOperationOrchestrator`, and
`ExecuteScheduledTaskOrchestrator`. `durabletaskscheduler.WithScheduledTasks()`
advertises the capability and keeps system orchestrators unversioned. The shared
host's registration-derived filters include these required handlers.

Every run owns a fresh schedule ID. A deferred cleanup retains the handle even
if creation is accepted but its wait fails. It first establishes creation, then
deletes the schedule and verifies absence, using a fresh 30-second cleanup
deadline while the worker is still running. Cleanup errors fail the command.
A finite **90-second `EndAt`** is a secondary safeguard, not a replacement for
verified deletion; its processing also requires a schedule worker.

Deletion stops future ticks, not already-started targets. Reports themselves are
finite single-activity workflows. Completed report and SDK operation history
remain for inspection. No broad purge, unrelated schedule deletion, or global
query is performed.

## Scheduling APIs and limitations

- The sample uses `Client.ScheduledTasks()`, `ScheduleClient`,
  `ScheduleCreationOptions`, and `ScheduleUpdateOptions`.
- The beta has no public `ScheduleClient.Run`/run-now API. “Run” here means
  observing real automatic recurring ticks after create/resume; it does not
  invoke private entity operations or manually schedule substitute reports.
- The command updates the interval and input and verifies the changed execution output.
- Payloads include the region, an ownership ID, and a phase.
  The default direct-target path is used (no retry, tags, or context wrapper),
  allowing queries to stay within the SDK-generated schedule-ID target prefix.
