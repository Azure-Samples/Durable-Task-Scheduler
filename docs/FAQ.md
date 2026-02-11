# Frequently Asked Questions

## General

**Q: What is the Durable Task Scheduler?**
A: The Durable Task Scheduler is a fully managed Azure service for durable execution — running fault-tolerant code that handles failures through automatic retries and state persistence. It serves as the backend for Durable Functions and the Durable Task SDKs. [Learn more →](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)

**Q: What's the difference between Durable Functions and Durable Task SDKs?**
A: Durable Functions is an extension of Azure Functions — best for serverless, event-driven apps with built-in triggers and auto-scaling. Durable Task SDKs are lightweight client libraries that work on any compute (Container Apps, AKS, VMs, etc.) — best when you need portability or already have a hosting environment. Both use the same Durable Task Scheduler backend. [See comparison →](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/choose-orchestration-framework)

**Q: What languages are supported?**
A: Durable Task SDKs: .NET, Python, Java (JavaScript coming soon). Durable Functions: .NET, Python, Java, JavaScript/TypeScript.

**Q: How much does it cost?**
A: The Durable Task Scheduler offers a Dedicated SKU (reserved capacity) and a Consumption SKU (preview, pay-per-use). The emulator is free for local development. [See pricing details →](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-dedicated-sku)

## Development

**Q: Can I develop locally without an Azure subscription?**
A: Yes! The Durable Task Scheduler emulator runs in Docker and provides the full experience including a monitoring dashboard. Just run: `docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest`

**Q: What is a Task Hub?**
A: A task hub is a logical container for orchestration and entity instances. You can create multiple task hubs within a single scheduler to isolate workloads by environment (dev/test/prod), team, or project. Each task hub gets its own monitoring dashboard. [Learn more →](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-task-hubs)

**Q: How does authentication work?**
A: The Durable Task Scheduler uses identity-based authentication only (Microsoft Entra ID / managed identity). No shared keys or connection string secrets. For local development with the emulator, no authentication is required. [Learn more →](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-identity)

**Q: Can I use the Durable Task Scheduler with existing Durable Functions apps?**
A: Yes! The Durable Task Scheduler is a backend provider for Durable Functions. You can switch from Azure Storage or other providers to the Durable Task Scheduler by updating your configuration. [Migration guide →](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler-functions)

## Monitoring & Observability

**Q: How do I monitor my orchestrations?**
A: The Durable Task Scheduler provides a built-in dashboard at [dashboard.durabletask.io](https://dashboard.durabletask.io) where you can view orchestration status, drill into execution history, and perform management operations (pause, terminate, restart). The emulator includes a local dashboard at http://localhost:8082.

**Q: Does it support distributed tracing?**
A: Yes. Durable Functions supports distributed tracing V2 with Application Insights. The Durable Task SDKs emit OpenTelemetry-compatible traces that can be exported to Jaeger, Zipkin, Application Insights, or any OTel-compatible backend.

## Troubleshooting

**Q: The emulator won't start**
A: Ensure Docker is running and ports 8080/8082 are free. Try: `docker pull mcr.microsoft.com/dts/dts-emulator:latest` to get the latest image. If ports conflict, map to different ports: `docker run -d -p 9080:8080 -p 9082:8082 mcr.microsoft.com/dts/dts-emulator:latest`

**Q: Where do I report bugs or request features?**
A: For issues with the Durable Task Scheduler service, use the [Azure/Durable-Task-Scheduler repo](https://github.com/Azure/Durable-Task-Scheduler/issues). For sample issues, open an issue in this repository.
