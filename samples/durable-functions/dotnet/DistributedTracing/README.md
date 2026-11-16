# Distributed Tracing with Durable Functions

.NET | Durable Functions

## Description

This sample demonstrates how to enable distributed tracing in Durable Functions using the Durable Task Scheduler backend. It shows the end-to-end trace correlation across orchestrators, activities, and HTTP triggers — viewable in both Jaeger (local) and Application Insights (Azure).

## Prerequisites

1. [.NET 10 SDK](https://dotnet.microsoft.com/download/dotnet/10.0) or later
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

The compose file starts Jaeger but does not configure the Functions host or worker to export telemetry to it. An OTLP exporter and endpoint configuration are also needed; starting Jaeger alone does not send it traces.

## Key Configuration

### HTTP authentication

`StartOrchestration` requires a function or host key in Azure. Core Tools skips key enforcement locally by default; use `func start --enableAuth` to exercise it locally.

For a deployed app, retrieve a non-admin host key into a shell variable and send it in a header:

```bash
FUNCTION_KEY=$(az functionapp keys list \
  --resource-group "$AZURE_RESOURCE_GROUP" \
  --name "$AZURE_FUNCTION_NAME" --query functionKeys.default --output tsv)
curl -X POST -H "x-functions-key: ${FUNCTION_KEY}" \
  "https://${AZURE_FUNCTION_NAME}.azurewebsites.net/api/StartOrchestration"
```

Do not use the master key or expose keys in source control, logs, or browser code. Keys do not provide per-user or per-instance authorization; managed identity for backend services is unchanged. The returned Durable management URLs contain a system key: keep those URLs secret and poll them as returned.

### Durable tracing

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

The Functions host uses OpenTelemetry and the Durable Task distributed tracing configuration above. When `APPLICATIONINSIGHTS_CONNECTION_STRING` is set, the worker exports telemetry to Azure Monitor and captures the `Microsoft.DurableTask` activity source to connect the starter to the orchestration trace. For user-assigned managed identity authentication, set `AzureMonitorExporter__credential` to `managedidentity` and `AzureMonitorExporter__clientId` to the identity's client ID, and grant that identity the Monitoring Metrics Publisher role on Application Insights.

1. Navigate to your Application Insights resource
2. Go to **Transaction Search**
3. Filter for events with Durable Functions prefixes (`orchestration:`, `activity:`)
4. Click on an event to see the Gantt chart showing the full orchestration flow

## Learn More

- [Observability Guide](../../../../docs/observability.md)
- [Durable Functions Diagnostics](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-diagnostics#distributed-tracing)
- [Durable Task Scheduler Dashboard](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-dashboard)
