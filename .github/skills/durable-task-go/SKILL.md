---
name: durable-task-go
description: Build durable workflows in Go with the standalone Durable Task SDK and Azure Durable Task Scheduler. Use for Go orchestrations, activities, entities, timers, events, schedules, versioning, payload/history extensions, or tracing. Does not apply to Durable Functions or Microsoft Agent Framework.
---

# Durable Task Go SDK with Durable Task Scheduler

Use Go **1.25.0+** and `github.com/microsoft/durabletask-go` **v1.0.0-beta.1**. This beta SDK targets DTS; do not substitute the older `durabletask-go` storage-backend APIs or generate Azure Functions bindings.

## Start from the samples

Read the [Go sample guide](../../../samples/durable-task-sdks/go) and the relevant sample before changing code. The samples share one module; do not create a nested `go.mod`.

With the emulator already running, from the repository root:

```bash
cd samples/durable-task-sdks/go
go mod download
go run ./function-chaining
```

Each package starts its worker and client together, runs a short demonstration, prints the result, and exits. Run other samples with `go run ./<sample-name>`.

## Keep samples readable

- Limit `main.go` to the entrypoint and CLI wiring.
- Keep orchestrations, activities, client code, and worker setup in focused files in the same sample package.
- Put domain types near their behavior. Avoid catch-all utility files, unnecessary interfaces, and extra package hierarchies.
- Make the default command demonstrate the pattern, not run an exhaustive test matrix.
- Put assertions and verification helpers in `*_test.go`; provide `TestIntegration` in `integration_test.go` using `testutil.IntegrationContext(t)`.
- Keep real input validation, operational errors, and safe cleanup in application code.
- Include a short README code map and standalone Go explanations.

## Connection and lifecycle

- Default to `Endpoint=http://localhost:8080;TaskHub=default;Authentication=None`, overridden by `DTS_CONNECTION_STRING`.
- For Azure, use `Endpoint=https://<scheduler-host>;TaskHub=<hub>;Authentication=DefaultAzure` or `Authentication=AzureCLI`. The identity needs Durable Task Data Contributor access. Never hard-code real resource identifiers or credentials.
- Parse the connection string with `durabletaskscheduler.NewOptionsFromConnectionString`.
- Register orchestrators and activities with `task.NewTaskRegistry`, `AddOrchestratorN`, and `AddActivityN`; check registration errors.
- Use `durabletaskscheduler.NewClient` for management and `durabletaskscheduler.NewWorker` for execution. `client.WithAutoWorkItemFilters()` routes work by registrations, allowing sample workers to share a hub.
- Start the worker before scheduling. Use bounded client contexts, inspect terminal status and output, close the client, and shut down the worker. Clean up only resources created by the sample.
- History-export requires a dedicated hub with no other export workers and no unrelated workloads completing during its export window; automatic work-item filters do not scope history-export queries. For both emulator and Azure runs, `HISTORY_EXPORT_ISOLATED_TASKHUB=1` acknowledges verified isolation but does not create or isolate a hub. Do not validate this sample alongside other samples on a shared hub.
- Preserve the sample's [implemented ownership guards](../../../samples/durable-task-sdks/go/history-export/ownership.go) and completion-window preflight. They reject unowned pages, metadata/history reads, and Blob writes and are covered by [offline unit tests](../../../samples/durable-task-sdks/go/history-export/main_test.go). These unit checks are not evidence of emulator or live backend validation.

The [released connection guide](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/durabletaskscheduler/README.md) is the authority for authentication, worker options, and feature-specific extension setup. Connecting a process to Azure is not deploying that process; these samples have no `azd` deployment templates.

## Replay-safe workflows

An orchestrator has signature `func(*task.OrchestrationContext) (any, error)`; an activity has signature `func(task.ActivityContext) (any, error)`.

- Use `ctx.CallActivity("Name", task.WithActivityInput(input)).Await(&result)` for activity results and handle errors.
- Keep HTTP, database access, filesystem access, random values, and environment reads out of orchestrators. Use activities, startup code, or supported context-bounded entity operations. External side effects can repeat if an entity operation fails before committing state.
- Use durable timers instead of `time.Sleep`; do not read the wall clock during replay.
- Use SDK tasks for concurrency, not native goroutines, channels, or `select` in orchestrators. Start all fan-out tasks before awaiting them.
- Keep ordering deterministic: sort map keys before scheduling work. Avoid mutable global state.
- Design activities for retries and possible duplicate execution; do not assume exactly-once side effects.
- Bound history with continue-as-new for recurring workflows. Sample demonstrations must still terminate.
- Use SDK entity operations and locks for durable state and coordination, not process-local mutexes. Check extension prerequisites for schedules, large payloads, and history export.
- Use numeric task versions such as `"1.0"` and `"2.0"` for DTS, even though the local registry accepts opaque version strings.

## Find the right pattern

| Need | Starting samples |
|------|------------------|
| Sequential or parallel work | [Function chaining](../../../samples/durable-task-sdks/go/function-chaining), [fan-out/fan-in](../../../samples/durable-task-sdks/go/fan-out-fan-in) |
| Wait for input or time | [Human interaction](../../../samples/durable-task-sdks/go/human-interaction), [monitoring](../../../samples/durable-task-sdks/go/monitoring) |
| Durable state | [Entities](../../../samples/durable-task-sdks/go/entities) |
| Recurring work | [Scheduled tasks](../../../samples/durable-task-sdks/go/scheduled-tasks), [bounded coordinator](../../../samples/durable-task-sdks/go/bounded-coordinator) |
| Reliability and evolution | [Saga](../../../samples/durable-task-sdks/go/saga), [versioning](../../../samples/durable-task-sdks/go/versioning), [testing](../../../samples/durable-task-sdks/go/testing) |
| Payloads and diagnostics | [Large payload](../../../samples/durable-task-sdks/go/large-payload), [history export](../../../samples/durable-task-sdks/go/history-export), [tracing](../../../samples/durable-task-sdks/go/opentelemetry-tracing) |

The [full catalog](../../../samples/README.md#go) and [pattern guide](../../../docs/patterns.md) cover the Go workflow patterns and integrations.

## Tracing

Configure an OpenTelemetry Go tracer provider/exporter and propagate the caller context when scheduling work. The Go SDK propagates W3C trace context; **DTS emits durable orchestration/activity/timer spans**. Do not claim the Go worker automatically exports those service spans locally. See the [observability guide](../../../docs/observability.md#go).

Set optional `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318` to export application spans over OTLP/HTTP to a running collector; this does not configure DTS service-side export. The integration tests separately check trace propagation and parentage.

## Validation

Format changed Go files with `gofmt`. From `samples/durable-task-sdks/go`:

```bash
go mod download
go build ./...
go test ./...
go vet ./...
```

The beta SDK has no public in-memory testing backend. `task.Executor` is exported for internal collaboration and uses `internal/protos` parameters; do not build application tests against it. Follow the testing sample's local step adapter to exercise shared business-workflow logic offline. These unit tests do not validate SDK execution or replay.

Ordinary tests must not contact a scheduler. Replay/integration tests require explicit `DTS_SAMPLES_E2E=1` and a real DTS emulator or Azure scheduler; never claim they passed when only offline checks ran. Read [contributor guidance](../../../CONTRIBUTING.md#go-samples) before running resource-backed tests.

For full-suite validation, prepare an isolated task hub and Blob endpoint, then run demos and their compiled `TestIntegration` binaries sequentially through `./e2e` rather than enabling integration tests across all packages concurrently:

```bash
HISTORY_EXPORT_ISOLATED_TASKHUB=1 DTS_SAMPLES_E2E=1 \
  go test -v -count=1 -timeout 30m ./e2e
```

Follow the [Go validation setup](../../../samples/durable-task-sdks/go/README.md#verify-every-sample-on-either-backend). The research demonstration uses synthetic fixtures by default, even with live DTS; these runs do not validate real model or arXiv services.
