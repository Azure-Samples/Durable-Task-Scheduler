# OpenTelemetry Distributed Tracing

| | |
|-|-|
| **Language** | C# (.NET 8) |
| **SDK** | Durable Task SDK |

This sample shows how to wire up [OpenTelemetry](https://opentelemetry.io/) with the Durable Task SDK to visualize orchestration flows in [Jaeger](https://www.jaegertracing.io/). It demonstrates trace correlation across an orchestrator, activities, and sub-orchestrations so you can see the full distributed trace of an order-processing workflow.

## Prerequisites

- [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
- [Docker](https://www.docker.com/products/docker-desktop)

## Quick Run

### 1. Start the infrastructure (DTS emulator + Jaeger)

```bash
docker compose up -d
```

This launches:
- **DTS Emulator** on `localhost:8080` (gRPC) and `localhost:8082` (Dashboard)
- **Jaeger** on `localhost:16686` (UI), `localhost:4317` (OTLP gRPC), and `localhost:4318` (OTLP HTTP)

### 2. Start the worker

```bash
dotnet run --project Worker
```

### 3. Run the client

In a separate terminal:

```bash
dotnet run --project Client
```

## Viewing Traces

1. Open the Jaeger UI at **http://localhost:16686**
2. Select the **durable-worker** service from the dropdown
3. Click **Find Traces**
4. Click on a trace to see the full span tree — you'll see the orchestrator span with child spans for each activity (`ValidateOrder`, `ProcessPayment`, `ShipOrder`, `SendNotification`)

You can also view the orchestration status in the DTS Dashboard at **http://localhost:8082**.

## How It Works

The worker registers the `Microsoft.DurableTask` activity source with OpenTelemetry and exports spans via OTLP to Jaeger. The Durable Task SDK automatically creates spans for orchestrations and activities, so you get distributed tracing out of the box.

Key configuration in `Worker/Program.cs`:

```csharp
builder.Services.AddOpenTelemetry()
    .ConfigureResource(resource => resource.AddService("durable-worker"))
    .WithTracing(tracing =>
    {
        tracing
            .AddSource("Microsoft.DurableTask")
            .AddOtlpExporter();
    });
```

## Clean Up

```bash
docker compose down
```

## Learn More

See [docs/observability.md](../../../../docs/observability.md) for more details on observability with the Durable Task Scheduler.
