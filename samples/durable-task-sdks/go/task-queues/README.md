# Task queues in Go

Run one simulated order with the standalone Durable Task Go SDK
`v1.0.0-beta3` and Go 1.25 or later. The parent starts with
`DTS_TaskQueue=orders`. Its acceptance activity inherits `orders`; a partner
activity and a child orchestration explicitly select `partner-api`. The
child's partner activity inherits `partner-api`. All task names start with
`GoTaskQueues` to avoid collisions with other languages.

## Prepare the task hub

Follow the [shared task queue setup](../../../../docs/task-queues.md) to create
`orders` at 20 handouts per second and `partner-api` at 5. Wait for both queues
to reach `Succeeded` before running against Azure. Queue tags do not create
queue resources. Missing or deleted queues use ordinary task hub routing
without the requested queue's rate limit.

Use a dedicated task hub and run only this worker in it. The worker is
unfiltered and registers every orchestration and activity it routes. It can
receive unrelated work in the same hub. Any work item filter, including a
wildcard, makes a connection ineligible for defined queues. The shared
`sample.Start` and `sample.WithHost` helpers enable automatic filters, so this
sample creates its worker directly.

The rates count work-item handouts, shared by orchestration turns and
activities in each queue. They do not guarantee completed orders or partner
requests per second. Bursts and redelivery affect the observed rate. Apply
request-level rate limiting in a real partner activity when strict limits
are required. The partner calls here have no external side effects.

## Run

From the repository root, with an existing Azure scheduler and dedicated hub:

```bash
cd samples/durable-task-sdks/go
export DTS_CONNECTION_STRING="Endpoint=https://<scheduler-host>;TaskHub=<dedicated-go-task-hub>;Authentication=AzureCLI"
go run ./task-queues -timeout 2m
```

Sign in with `az login`. The identity needs Durable Task Data Contributor for
data-plane access; queue provisioning needs management permissions. Use
`Authentication=DefaultAzure` for managed identity or default Azure credentials.
See the [Go connection guide](../README.md#connect-to-azure-dts).

With a running emulator, the default connection is
`Endpoint=http://localhost:8080;TaskHub=default;Authentication=None`.
Use only this worker in the emulator hub as well. The emulator can exercise
tags and workflow completion; this sample does not validate Azure queue
throttling against the emulator.

The command checks the final runtime status and all result fields, prints the
instance ID, completed status, and order result, then stops the worker and
closes the client. On failure it attempts to terminate its unfinished parent
and children. Completed instances remain available in the dashboard.

## Tests

From `samples/durable-task-sdks/go`:

```bash
go build -o /dev/null ./task-queues
go test ./task-queues
go vet ./task-queues
DTS_SAMPLES_E2E=1 go test -v -count=1 -run '^TestIntegration$' ./task-queues
```

Ordinary tests run offline and check activity input handling and result
validation. They do not check orchestration replay. `TestIntegration` uses
`testutil.IntegrationContext` and checks completed outputs, parent and child
metadata, and activity/child queue tags in history. It skips unless
`DTS_SAMPLES_E2E=1`. Use an emulator or Azure scheduler with only this worker.
On Azure, provision both queues first to exercise defined-queue routing.
The integration test does not measure handout rates on either backend.

## Code map

| File | Purpose |
|---|---|
| `main.go` | Entry point and shared timeout flag |
| `workflow.go` | Parent and child orchestrations, input and result types |
| `activities.go` | Simulated acceptance and partner calls |
| `worker.go` | Registration and unfiltered worker lifecycle |
| `client.go` | Queue-tagged start, completion checks, output, failure cleanup |
| `main_test.go` | Offline activity and output checks |
| `integration_test.go` | Opt-in completion and persisted-tag checks |

See [task queues in DTS](https://learn.microsoft.com/en-us/azure/durable-task/scheduler/durable-task-scheduler-task-queues)
and the [Go SDK connection reference](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta3/durabletaskscheduler/README.md).
