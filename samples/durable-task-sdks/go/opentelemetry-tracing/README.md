# OpenTelemetry distributed tracing (Go)

A sample order goes through validation, payment, shipping, and notification.
The notification step calls a local HTTP test service. The demo prints the
result and the trace ID. You can send application spans to Jaeger or another
OTLP/HTTP collector. (A span is one operation in a trace.)

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup. No
collector is needed for the default demo.

## Run

From this directory:

```bash
go run . -timeout 3m
```

Or, from the Go samples directory: `go run ./opentelemetry-tracing -timeout 3m`.

Example output:

```text
Processing synthetic order Order-12345
Result: Notified(Shipped(Paid(Validated(Order-12345))))
Trace ID: ...
Instance: go-tracing-...
Set OTEL_EXPORTER_OTLP_ENDPOINT to visualize application spans.
```

### Optional: view spans in Jaeger

The [compose file](docker-compose.yml) starts **only Jaeger**:

```bash
docker compose up -d
export OTEL_EXPORTER_OTLP_ENDPOINT='http://localhost:4318'
go run . -timeout 3m
```

Open <http://localhost:16686> and select **GoOrderProcessingSample**, or search
for the trace ID. Use port **4318** (OTLP/HTTP), not 4317 (OTLP/gRPC).

| Variable | Behavior |
| --- | --- |
| Neither endpoint set | Spans are created but not exported |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Base URL; the exporter adds `/v1/traces` |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Full traces URL; overrides the base URL |

Use HTTPS for a remote collector. Export errors cause a nonzero exit code.
Stop Jaeger with `docker compose down`.

## How spans are connected

The client starts a span and passes its trace context to DTS. **DTS creates the
orchestration, activity, and timer spans.** The Go SDK gives that context to
activities through `ActivityContext.Context()`, but it does not export the DTS
spans locally. The app creates a child span for each activity and sends the
trace context to the HTTP service with W3C headers. Because DTS spans are not
exported locally, some parent spans do not appear in Jaeger.

## Code map

| File | Purpose |
| --- | --- |
| [main.go](main.go) | Command-line entry point |
| [workflow.go](workflow.go) | The four-step order orchestration |
| [activities.go](activities.go) | Order steps |
| [client.go](client.go) | Schedules the orchestration under a caller span |
| [worker.go](worker.go) | Registers the orchestration and traced activities |
| [telemetry.go](telemetry.go) | Tracer setup, optional OTLP export, and shutdown |
| [notification.go](notification.go) | Local HTTP service and trace headers |
| [integration_test.go](integration_test.go), [verify_test.go](verify_test.go), [observations_test.go](observations_test.go) | Span and history checks |

## Tests

```bash
go test -mod=readonly .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The integration test adds an in-memory exporter. It checks results, trace IDs,
and parent-child links between spans.

See the [SDK tracing sample](https://github.com/microsoft/durabletask-go/tree/v1.0.0-beta3/samples/distributedtracing)
and the [OpenTelemetry Go documentation](https://opentelemetry.io/docs/languages/go/).
