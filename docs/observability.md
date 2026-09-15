# Observability & Distributed Tracing

Monitoring and debugging distributed workflows is critical for production systems. The Durable Task Scheduler ecosystem provides multiple layers of observability: a built-in dashboard for business-level monitoring and OpenTelemetry support for infrastructure-level tracing.

## Table of Contents
- Built-in Dashboard
- Distributed Tracing with OpenTelemetry
- Durable Functions Distributed Tracing
- Durable Task SDKs Tracing
- Exporter Options
- Next Steps

---

## Built-in Dashboard

Every Durable Task Scheduler instance (including the local emulator) comes with a monitoring dashboard out of the box.

### What you can do:
- **View all orchestrations** — filter by status, name, time range
- **Drill into execution history** — see each activity, sub-orchestration, and event
- **Monitor timing** — identify slow activities and bottlenecks
- **Manage instances** — pause, terminate, restart, or purge orchestration instances
- **Multi-agent visualization** — trace complex AI agent workflows across multiple orchestrations

### Access the dashboard:

**Local (Emulator):**
```
http://localhost:8082
```

**Azure:**
Navigate to your Durable Task Scheduler resource → Task Hub → Dashboard URL, or go to [dashboard.durabletask.io](https://dashboard.durabletask.io) and register your endpoint.

📖 [Dashboard documentation →](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-dashboard)

---

## Distributed Tracing with OpenTelemetry

For infrastructure-level observability — latency analysis, cross-service correlation, and performance profiling — you can use OpenTelemetry (OTel) distributed tracing.

### How it complements the dashboard

| Aspect | Built-in Dashboard | OpenTelemetry Tracing |
|--------|-------------------|----------------------|
| **Focus** | Business logic (orchestration state) | Infrastructure (latency, errors, dependencies) |
| **Granularity** | Orchestration/activity level | Span-level (including HTTP, DB calls) |
| **Cross-service** | Within task hub | Across all services (end-to-end) |
| **Storage** | Managed by DTS | Your choice (App Insights, Jaeger, etc.) |
| **Best for** | "What happened in this orchestration?" | "Where is the bottleneck across my system?" |

---

## Durable Functions Distributed Tracing

Durable Functions supports **Distributed Tracing V2**, which correlates orchestrations, entities, and activities into unified traces.

### Setup

1. **Update host.json:**
```json
{
  "extensions": {
    "durableTask": {
      "tracing": {
        "distributedTracingEnabled": true,
        "version": "V2"
      }
    }
  }
}
```

2. **Requirements:**
   - .NET Isolated: `Microsoft.Azure.Functions.Worker.Extensions.DurableTask` >= v1.4.0
   - Non-.NET: `Microsoft.Azure.WebJobs.Extensions.DurableTask` >= v3.2.0

3. **Configure Application Insights** — If your Function app has Application Insights enabled, traces will appear automatically.

### Viewing traces in Application Insights

1. Navigate to your Application Insights resource
2. Go to **Transaction Search**
3. Filter for `Request` and `Dependency` events with Durable Functions prefixes (`orchestration:`, `activity:`)
4. Click on an event to see the end-to-end Gantt chart

The Gantt chart shows the full orchestration flow — when each activity started, how long it took, and the data flow between them.

📖 [Durable Functions diagnostics →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-diagnostics#distributed-tracing)

---

## Durable Task SDKs Tracing

The Durable Task SDKs integrate with OpenTelemetry. Which spans are emitted locally versus by DTS depends on the SDK; configure the tracer provider and exporter for your language.

### .NET

Add OpenTelemetry packages to your project:

```xml
<PackageReference Include="OpenTelemetry" Version="1.*" />
<PackageReference Include="OpenTelemetry.Extensions.Hosting" Version="1.*" />
<PackageReference Include="OpenTelemetry.Exporter.OtlpProtocol" Version="1.*" />
```

Configure tracing in your worker:

```csharp
builder.Services.AddOpenTelemetry()
    .WithTracing(tracing =>
    {
        tracing
            .AddSource("Microsoft.DurableTask")
            .AddOtlpExporter(opts =>
            {
                opts.Endpoint = new Uri("http://localhost:4317");
            });
    });
```

### Python

Install OpenTelemetry packages:

```bash
pip install opentelemetry-api opentelemetry-sdk opentelemetry-exporter-otlp
```

Configure tracing:

```python
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

provider = TracerProvider()
processor = BatchSpanProcessor(OTLPSpanExporter(endpoint="http://localhost:4317"))
provider.add_span_processor(processor)
trace.set_tracer_provider(provider)
```

### Go

The Go SDK (`github.com/microsoft/durabletask-go` v1.0.0-beta.1, Go 1.25.0+) propagates **W3C trace context** from the caller through DTS to activities. Configure an OpenTelemetry tracer provider and exporter in your application, start a caller span, and pass that context when scheduling an orchestration. Activities can create application or dependency spans using the propagated context.

**DTS owns the durable orchestration, activity, and timer spans.** The Go worker does not duplicate these service spans in your local exporter. Seeing application spans or matching trace IDs in orchestration history verifies propagation, not export of the full service-side trace.

Start with the [Go OpenTelemetry sample](../samples/durable-task-sdks/go/opentelemetry-tracing):

```bash
cd samples/durable-task-sdks/go
go mod download
go run ./opentelemetry-tracing
```

Run the emulator first and follow that sample's README for its tracing configuration. The demo shows a traced workflow; its opt-in integration tests verify application spans and trace parentage separately.

To also export application spans to a running OTLP/HTTP collector or Jaeger, set the optional endpoint from the same Go module:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 go run ./opentelemetry-tracing
```

Use HTTP port **4318**, not OTLP/gRPC port 4317. An explicitly configured but unavailable collector fails the sample. Setting this endpoint configures the sample's application exporter, not export of DTS-owned durable spans.

See the [released SDK tracing example](https://github.com/microsoft/durabletask-go/tree/v1.0.0-beta.1/samples/distributedtracing) and [OpenTelemetry Go documentation](https://opentelemetry.io/docs/languages/go/) for exporter setup.

---

## Local Development with Jaeger

For local development, you can use Jaeger to visualize traces alongside the DTS emulator.

### Docker Compose setup

```yaml
version: '3.8'
services:
  dts-emulator:
    image: mcr.microsoft.com/dts/dts-emulator:latest
    ports:
      - "8080:8080"
      - "8082:8082"

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "4317:4317"    # OTLP gRPC
      - "4318:4318"    # OTLP HTTP
    environment:
      - COLLECTOR_OTLP_ENABLED=true
```

After starting both services:
- **DTS Dashboard:** http://localhost:8082
- **Jaeger UI:** http://localhost:16686

---

## Exporter Options

| Exporter | Best For | Setup Complexity |
|----------|----------|-----------------|
| **Application Insights** | Azure production workloads | Low (built-in for Azure Functions) |
| **Jaeger** | Local development, self-hosted | Low (Docker) |
| **Zipkin** | Lightweight tracing | Low (Docker) |
| **Grafana Tempo** | Grafana ecosystem users | Medium |
| **OTLP (generic)** | Any OTel-compatible backend | Varies |

For Azure production workloads in supported languages, use **Application Insights** with the [Azure Monitor OpenTelemetry Distro](https://learn.microsoft.com/azure/azure-monitor/app/opentelemetry-enable). Check its language support before choosing an exporter; the distro setup is not a Go SDK integration. For Go, configure an OpenTelemetry Go exporter and an appropriate collector/backend.

---

## Next Steps

- [Durable Functions Diagnostics →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-diagnostics)
- [.NET Observability with OpenTelemetry →](https://learn.microsoft.com/dotnet/core/diagnostics/observability-with-otel)
- [Go OpenTelemetry Sample →](../samples/durable-task-sdks/go/opentelemetry-tracing)
- [OpenTelemetry on Azure →](https://learn.microsoft.com/azure/azure-monitor/app/opentelemetry)
- [Dashboard Documentation →](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-dashboard)
