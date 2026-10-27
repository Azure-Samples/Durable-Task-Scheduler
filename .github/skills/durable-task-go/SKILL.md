---
name: durable-task-go
description: Build durable workflows in Go with the standalone Durable Task SDK and Azure Durable Task Scheduler. Use for Go orchestrations, activities, entities, timers, events, schedules, versioning, payload/history extensions, or tracing. Does not apply to Durable Functions or Microsoft Agent Framework.
---

# Durable Task Go SDK with Durable Task Scheduler

Use Go **1.25.0+** and `github.com/microsoft/durabletask-go` **v1.0.0-beta3**. This SDK connects to DTS. Do not use the older `durabletask-go` storage-backend APIs, and do not generate Azure Functions bindings.

## Start from the samples

Read the [Go sample guide](../../../samples/durable-task-sdks/go) and the relevant sample before you change code. All samples share one module; do not create a nested `go.mod`.

With the emulator running, from the repository root:

```bash
cd samples/durable-task-sdks/go
go mod download
go run ./function-chaining
```

Each sample starts its worker and client, runs a short demo, prints the result, and exits. Run other samples with `go run ./<sample-name>`.

## Sample structure

- `main.go` has only the entry point and CLI flags.
- Put orchestrations, activities, client code, and worker setup in separate files in the same package.
- Keep types near the code that uses them. Avoid catch-all utility files, extra interfaces, and extra packages.
- The default command shows the pattern. It does not run a full test matrix.
- Put assertions in `*_test.go`. Add `TestIntegration` in `integration_test.go` using `testutil.IntegrationContext(t)`.
- Keep input validation, error handling, and cleanup in app code.
- Add a short README with a code map.

## Connection and lifecycle

- Default connection: `Endpoint=http://localhost:8080;TaskHub=default;Authentication=None`. `DTS_CONNECTION_STRING` overrides it.
- Azure: `Endpoint=https://<scheduler-host>;TaskHub=<hub>;Authentication=DefaultAzure` or `Authentication=AzureCLI`. The identity needs the Durable Task Data Contributor role. Never hard-code real resource IDs or credentials.
- Parse the connection string with `durabletaskscheduler.NewOptionsFromConnectionString`.
- Register tasks with `task.NewTaskRegistry`, `AddOrchestratorN`, and `AddActivityN`. Check registration errors.
- Use `durabletaskscheduler.NewClient` for management and `durabletaskscheduler.NewWorker` for execution. `client.WithAutoWorkItemFilters()` routes work by registration, so sample workers can share a hub.
- Start the worker before you schedule work. Use client contexts with timeouts, check the final status and output, close the client, and stop the worker. Clean up only resources the sample created.
- History export needs a separate hub with no other export workers and no unrelated work completing during the export window. Work-item filters do not limit export queries. `HISTORY_EXPORT_ISOLATED_TASKHUB=1` confirms isolation; it does not create it.
- Keep the history-export [ownership guards](../../../samples/durable-task-sdks/go/history-export/ownership.go) and their [unit tests](../../../samples/durable-task-sdks/go/history-export/main_test.go). Unit tests are not proof of backend validation.

The [SDK connection guide](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta3/durabletaskscheduler/README.md) is the reference for authentication, worker options, and extensions. The samples have no `azd` deployment templates.

## Replay-safe orchestrations

Orchestrator signature: `func(*task.OrchestrationContext) (any, error)`. Activity signature: `func(task.ActivityContext) (any, error)`.

- Call activities with `ctx.CallActivity("Name", task.WithActivityInput(input)).Await(&result)` and handle the error.
- No HTTP, database, file, random, or environment access in orchestrators. Use activities or entity operations. Entity side effects can repeat if the operation fails before it saves state.
- Use durable timers, not `time.Sleep`. Do not read the wall clock.
- Use SDK tasks for concurrency, not goroutines, channels, or `select`. Start all fan-out tasks before you wait for them.
- Keep order deterministic: sort map keys before scheduling work. Avoid mutable global state.
- Activities can run more than once. Do not assume exactly-once side effects.
- Use continue-as-new to limit history in recurring orchestrations. Sample demos must still end.
- Use entities and entity locks for durable state, not process mutexes.
- Use numeric versions such as `"1.0"` and `"2.0"` for DTS.

## Find the right pattern

| Need | Samples |
|------|---------|
| Sequential or parallel work | [Function chaining](../../../samples/durable-task-sdks/go/function-chaining), [fan-out/fan-in](../../../samples/durable-task-sdks/go/fan-out-fan-in) |
| Wait for input or time | [Human interaction](../../../samples/durable-task-sdks/go/human-interaction), [monitoring](../../../samples/durable-task-sdks/go/monitoring) |
| Durable state | [Entities](../../../samples/durable-task-sdks/go/entities) |
| Recurring work | [Scheduled tasks](../../../samples/durable-task-sdks/go/scheduled-tasks), [bounded coordinator](../../../samples/durable-task-sdks/go/bounded-coordinator) |
| Reliability and change | [Saga](../../../samples/durable-task-sdks/go/saga), [versioning](../../../samples/durable-task-sdks/go/versioning), [testing](../../../samples/durable-task-sdks/go/testing) |
| Payloads and diagnostics | [Large payload](../../../samples/durable-task-sdks/go/large-payload), [history export](../../../samples/durable-task-sdks/go/history-export), [tracing](../../../samples/durable-task-sdks/go/opentelemetry-tracing) |

See the [sample catalog](../../../samples/README.md#go) and [pattern guide](../../../docs/patterns.md).

## Tracing

Set up an OpenTelemetry Go tracer provider and pass the caller context when you schedule work. The SDK propagates W3C trace context. **DTS creates the orchestration, activity, and timer spans**; the Go worker does not export them locally. See the [observability guide](../../../docs/observability.md#go).

`OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318` exports application spans over OTLP/HTTP. It does not change DTS export.

## Validation

Format changed files with `gofmt`. From `samples/durable-task-sdks/go`:

```bash
go mod download
go build ./...
go test ./...
go vet ./...
```

The SDK has no public in-memory test backend. Do not build tests on `task.Executor`; it uses internal types. Use the testing sample's local step adapter for offline business-logic tests. These tests do not check replay.

Ordinary tests must not contact a scheduler. Integration tests need `DTS_SAMPLES_E2E=1` and a real emulator or Azure scheduler. Never say they passed if only offline checks ran. See the [contributor guide](../../../CONTRIBUTING.md#go-samples).

For the full suite, prepare a separate task hub and Blob endpoint, then run the sequential runner:

```bash
HISTORY_EXPORT_ISOLATED_TASKHUB=1 DTS_SAMPLES_E2E=1 \
  go test -v -count=1 -timeout 30m ./e2e
```

See the [Go validation setup](../../../samples/durable-task-sdks/go/README.md#verify-every-sample-on-either-backend).
