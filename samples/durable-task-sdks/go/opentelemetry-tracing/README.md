# OpenTelemetry distributed tracing

Go | Durable Task SDK

## Description

This counterpart to the [Python order-processing sample](../../python/opentelemetry-tracing/)
runs the same chain: **validate → pay → ship → notify**.
One command starts a filtered DTS worker and client and validates trace propagation
without requiring an external telemetry service.

The Go SDK's tracing model differs from Python's:

- The caller starts a **valid, sampled** OpenTelemetry span and passes its context
  to `ScheduleNewOrchestration`.
- **DTS owns durable orchestration/activity/timer spans.** The Go worker restores
  a non-recording remote context into `ActivityContext.Context()`; it does **not**
  duplicate those durable spans in the local tracer provider.
- The application explicitly creates four user activity spans from that context.
  Notification performs a real loopback HTTP request with an outbound client span,
  W3C header injection/extraction, and a server span.
- A per-run provider and explicit propagator avoid global provider/propagator
  mutation. All seven application spans are captured in memory for verification.

The orchestrator itself creates no user spans or nondeterministic telemetry during
replay. User work and outbound I/O are instrumented inside activities.

## Prerequisites

- Go 1.25+.
- A running DTS emulator or existing live scheduler/task hub. Follow the
  [shared emulator/live authentication setup](../README.md).
- No Blob storage, real orders, notification service, or telemetry infrastructure
  is needed for default execution. The HTTP target is an ephemeral loopback server
  owned by this process.

## Run

From `samples/durable-task-sdks/go`:

```bash
go run ./opentelemetry-tracing
```

## Optional Jaeger visualization / real OTLP export

The compose file starts **only Jaeger**, without changing the shared DTS emulator:

```bash
docker compose -f opentelemetry-tracing/docker-compose.yml up -d
export OTEL_EXPORTER_OTLP_ENDPOINT='http://localhost:4318'
go run ./opentelemetry-tracing
```

Open <http://localhost:16686>, select service **GoOrderProcessingSample**, or search
for the printed trace ID. Use OTLP/**HTTP** port **4318**, not the gRPC port 4317.
For a different collector use HTTPS as appropriate. The official OTLP HTTP exporter
supports standard headers/certificate variables; supply credentials securely.

| Variable | Meaning |
| --- | --- |
| Neither endpoint variable set | In-memory verification only; no OTLP connection attempted |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP/HTTP base URL, e.g. `http://localhost:4318` (exporter appends `/v1/traces`) |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Full traces URL, e.g. `http://localhost:4318/v1/traces`; takes precedence |

An explicitly configured, unavailable collector **fails** the command. Exporter
errors are retained even if they occurred in an earlier asynchronous batch.
`ForceFlush` and `Shutdown` are awaited and their errors surface before `SAMPLE_OK`.
A successful OTLP response proves collector acceptance, not downstream Jaeger
retention/query availability.

### Optional service-owned spans

To visualize **DTS-owned** spans too, separately configure your DTS
emulator/deployment's supported backend tracing integration to export to the same
collector. This is optional, deployment-specific, and **not changed by this sample**.
Configuring the worker's OTLP endpoint does not configure managed DTS.

Without service telemetry, Jaeger contains only the explicit application spans;
some user spans reference remote parents absent from the collector. That is
expected, not a reason to manufacture local orchestration/activity spans.
The default checks validate persisted DTS trace **contexts**, not receipt of
DTS service spans in the in-memory exporter.

## Expected output and assertions

```text
Result: Notified(Shipped(Paid(Validated(Order-12345))))
Trace ID: ...; instance: go-tracing-...
Verified sampled caller, 4 durable activity trace contexts, 4 user activity spans, and HTTP client/server propagation
Application spans verified in memory; set OTEL_EXPORTER_OTLP_ENDPOINT for Jaeger
SAMPLE_OK opentelemetry-tracing
```

Success requires:

1. A completed order with all four expected intermediate outputs.
2. Every activity observes a valid, sampled, **remote, non-recording** SDK context.
3. An execution-ID-pinned history contains the original caller context and all
   four scheduled activities' valid W3C contexts with the same trace ID.
4. The in-memory exporter receives the caller and each user span under that trace;
   the user spans' parents match the remote activity contexts.
5. The HTTP server observes the same trace ID and the actual outbound client span
   as its remote parent, and the exporter records that relationship.
6. No activity, export, flush, or shutdown failure is ignored.

Use `-timeout 5m` for a slow DTS environment. Tests use only in-process contexts,
an in-memory exporter, and a loopback test HTTP server:

```bash
go test -mod=readonly ./opentelemetry-tracing
```

## Cleanup

The command closes its worker, client, HTTP server, and tracer provider. The
completed orchestration remains available in the DTS dashboard. Stop only the
optional Jaeger compose project when finished:

```bash
docker compose -f opentelemetry-tracing/docker-compose.yml down
```

## API references

- [Released Go distributed tracing sample](https://github.com/microsoft/durabletask-go/tree/v1.0.0-beta.1/samples/distributedtracing)
- [ActivityContext public API](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/task/activity.go)
- [SDK trace restoration test](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/task/activity_trace_test.go)
- [OpenTelemetry Go documentation](https://opentelemetry.io/docs/languages/go/)

The upstream tracing example is a nested module. This sample instead uses the
shared Go module at `../go.mod`; do not initialize a module in this directory.
