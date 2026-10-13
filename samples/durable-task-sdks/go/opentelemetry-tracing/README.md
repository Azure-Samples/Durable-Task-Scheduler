# OpenTelemetry distributed tracing

Go | Durable Task SDK

Trace a sample order through validation, payment, shipping, and notification.
The notification step calls a local HTTP test service, not a customer service.
The demo prints the result and trace ID. You can also use an OTLP/HTTP exporter
to send application spans to Jaeger or another collector.
A span records one operation within a trace.

## Prerequisites

- Go 1.25+ and the [shared emulator/live DTS setup](../README.md).
- No Blob storage, external notification service, or collector is required for
  the default demo.

## Run

From this directory:

```bash
go run . -timeout 3m
```

From the shared module root, use `go run ./opentelemetry-tracing -timeout 3m`.
All sample data is in the code. You can also run the compiled program from
either directory.

Example output:

```text
Processing synthetic order Order-12345
Result: Notified(Shipped(Paid(Validated(Order-12345))))
Trace ID: ...
Instance: go-tracing-...
Set OTEL_EXPORTER_OTLP_ENDPOINT to visualize application spans.
```

### Optional Jaeger

The [compose file](docker-compose.yml) starts **only Jaeger**, leaving your DTS
configuration unchanged:

```bash
docker compose up -d
export OTEL_EXPORTER_OTLP_ENDPOINT='http://localhost:4318'
go run . -timeout 3m
```

Open <http://localhost:16686> and select **GoOrderProcessingSample**, or search for
the printed trace ID. Port **4318** is OTLP/HTTP, not OTLP/gRPC.

| Variable | Behavior |
| --- | --- |
| Neither endpoint set | Trace context and spans are created but are not sent to a collector |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Base URL; the exporter appends `/v1/traces` |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Full traces URL; overrides the base URL |

Use HTTPS when connecting to a remote collector. The exporter supports the
standard OTLP environment settings for headers and certificates.
The command reports exporter errors when it sends remaining spans or shuts down.
This includes errors from earlier background exports. Such errors cause a
nonzero exit status. A collector accepting spans does not prove that they are
stored or visible in a tracing tool.

## How spans are connected

The client gives DTS a trace context marked for recording.
**DTS creates the durable orchestration, activity, and timer spans.**
The Go SDK restores their context in `ActivityContext.Context()` without creating
local copies of those service spans. The application creates a child span
around each activity and passes trace information through W3C HTTP headers.

The code passes tracing providers and context handlers as arguments instead of
using global settings. The orchestrator creates no spans during replay.
To view DTS service spans too, set up the service's tracing support separately.
Without this setup, some parent spans will not appear in Jaeger.
The sample does not change the scheduler's tracing settings.

## Code map

| File | Responsibility |
| --- | --- |
| [main.go](main.go) | Starts the command-line program and sets its timeout |
| [workflow.go](workflow.go) | The four-step durable order chain |
| [activities.go](activities.go) | Synthetic order stages |
| [client.go](client.go) | Schedule under a caller span and display the result |
| [worker.go](worker.go) | Register the workflow and instrumented activities |
| [telemetry.go](telemetry.go) | Sets up tracing, optional OTLP export, and safe shutdown |
| [notification.go](notification.go) | Local HTTP test service and trace-context handling |
| [integration_test.go](integration_test.go), [verify_test.go](verify_test.go) | Checks history for specific executions, results, and spans |
| [observations_test.go](observations_test.go) | Records trace contexts for tests |

## Testing

Offline tests use a local HTTP test service and an in-memory exporter:

```bash
go test -mod=readonly .
```

With the configured DTS backend available:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The integration test adds an in-memory exporter to the application's provider.
It runs the same workflow, activities, and HTTP handler as the demo.
The test checks each step's result and the history for the expected execution ID.
It also checks trace IDs, recording flags, remote contexts, and parent-child
links between application and HTTP spans.
The in-memory exporter and detailed checks are **not part of the runnable demo**.

## Cleanup

The worker, client, local HTTP service, and tracer provider close automatically. The
completed orchestration remains in the DTS dashboard. Stop the optional Jaeger
compose project with `docker compose down` when finished.

[Go tracing sample](https://github.com/microsoft/durabletask-go/tree/v1.0.0-beta.1/samples/distributedtracing)
and [OpenTelemetry Go documentation](https://opentelemetry.io/docs/languages/go/).
