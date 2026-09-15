# OpenTelemetry distributed tracing

Go | Durable Task SDK

Trace a synthetic order through validation, payment, shipping, and notification.
The notification step calls a loopback HTTP fixture, not a customer service.
The demo prints its business result and trace ID; an optional OTLP/HTTP exporter
sends application spans to Jaeger or another collector.

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
All fixture data is in code; the demo also works as a compiled binary from either
directory.

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
| Neither endpoint set | Context/spans are created; no exporter is configured |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Base URL; the exporter appends `/v1/traces` |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Full traces URL; overrides the base URL |

Use HTTPS for a remote collector as appropriate. The official exporter supports
standard OTLP header/certificate environment settings. Configured exporter
failures, including earlier asynchronous failures, surface during flush/shutdown
and make the command exit nonzero. Collector acceptance does not prove retention
or query availability in a downstream tracing UI.

## How spans are connected

The client supplies a sampled caller context to DTS. **DTS owns durable
orchestration/activity/timer spans**; the Go SDK restores their remote context in
`ActivityContext.Context()` without duplicating those service spans locally.
Application instrumentation creates a child span around each activity and
propagates W3C headers across the HTTP request.

Providers and propagators are passed explicitly, not installed globally.
The orchestrator creates no spans during replay. To visualize DTS-owned spans
as well, configure the service's supported backend tracing integration separately.
Without it, some application spans reference remote parents absent from Jaeger.
This sample does not change the scheduler's telemetry configuration.

## Code map

| File | Responsibility |
| --- | --- |
| [main.go](main.go) | Entrypoint and shared timeout |
| [workflow.go](workflow.go) | The four-step durable order chain |
| [activities.go](activities.go) | Synthetic order stages |
| [client.go](client.go) | Schedule under a caller span and display the result |
| [worker.go](worker.go) | Register the workflow and instrumented activities |
| [telemetry.go](telemetry.go) | Provider, custom activity spans, optional OTLP, error-aware shutdown |
| [notification.go](notification.go) | Loopback HTTP fixture and trace-context propagation |
| [integration_test.go](integration_test.go), [verify_test.go](verify_test.go) | Pinned history, output, and span assertions |
| [observations_test.go](observations_test.go) | Test-only observation of contexts passed to the real instrumentation |

## Testing

Offline tests use a local HTTP fixture and an in-memory exporter:

```bash
go test -mod=readonly .
```

With the configured DTS backend available:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The integration test attaches an in-memory exporter to the production provider
and runs the same production workflow, activities, and HTTP handler. It checks
all intermediate/final order results, execution-ID-pinned history, sampled caller
propagation, non-recording remote activity contexts, user span parentage, and
outbound HTTP/server topology. An in-memory exporter and these exhaustive
assertions are **not part of the runnable demo**.

## Cleanup

The worker, client, HTTP fixture, and tracer provider close automatically. The
completed orchestration remains in the DTS dashboard. Stop the optional Jaeger
compose project with `docker compose down` when finished.

[Go tracing sample](https://github.com/microsoft/durabletask-go/tree/v1.0.0-beta.1/samples/distributedtracing)
and [OpenTelemetry Go documentation](https://opentelemetry.io/docs/languages/go/).
