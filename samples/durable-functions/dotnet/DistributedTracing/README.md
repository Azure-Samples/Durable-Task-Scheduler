# Distributed Tracing with Durable Functions

.NET | Durable Functions

## Description

This sample demonstrates how to enable distributed tracing in Durable Functions using the Durable Task Scheduler backend. It shows the end-to-end trace correlation across orchestrators, activities, and HTTP triggers — viewable in both Jaeger (local) and Application Insights (Azure).

## Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0) or later
2. [Docker](https://www.docker.com/products/docker-desktop/)
3. [Azure Functions Core Tools](https://learn.microsoft.com/azure/azure-functions/functions-run-local) v4+

## Quick Run (Local with Jaeger)

1. Start the infrastructure:
   ```bash
   docker compose up -d
   ```

2. Run the function app:
   ```bash
   func start
   ```

3. Trigger an orchestration:
   ```bash
   curl -X POST http://localhost:7071/api/StartOrchestration
   ```

4. View traces:
   - **Jaeger UI:** http://localhost:16686
   - **DTS Dashboard:** http://localhost:8082

## Key Configuration

Distributed tracing is enabled in `host.json`:

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

## Using with Application Insights (Azure)

When deployed to Azure with Application Insights configured:

1. Navigate to your Application Insights resource
2. Go to **Transaction Search**
3. Filter for events with Durable Functions prefixes (`orchestration:`, `activity:`)
4. Click on an event to see the Gantt chart showing the full orchestration flow

## Learn More

- [Observability Guide](../../../../docs/observability.md)
- [Durable Functions Diagnostics](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-diagnostics#distributed-tracing)
- [Durable Task Scheduler Dashboard](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-dashboard)
