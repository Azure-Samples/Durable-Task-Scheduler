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

The Durable Task SDKs emit traces that can be collected using OpenTelemetry.

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

For Azure production workloads, we recommend **Application Insights** with the [Azure Monitor OpenTelemetry Distro](https://learn.microsoft.com/azure/azure-monitor/app/opentelemetry-enable).

---

## Next Steps

- [Durable Functions Diagnostics →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-diagnostics)
- [.NET Observability with OpenTelemetry →](https://learn.microsoft.com/dotnet/core/diagnostics/observability-with-otel)
- [OpenTelemetry on Azure →](https://learn.microsoft.com/azure/azure-monitor/app/opentelemetry)
- [Dashboard Documentation →](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-dashboard)
