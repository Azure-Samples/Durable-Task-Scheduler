# 📚 Sample Catalog

Explore a comprehensive collection of samples for building durable, fault-tolerant workflows with **Azure Durable Task Scheduler**. Whether you're just getting started or building advanced AI agents, there's a sample for you.

> **Prerequisites:** Most samples require [Docker](https://www.docker.com/) to run the Durable Task Scheduler emulator locally. See each sample's README for specific setup instructions.

---

## 🚀 Start Here

New to Durable Task Scheduler? Start with the **Function Chaining** sample in your preferred language:

| Language | Sample | Description |
|----------|--------|-------------|
| .NET | [Function Chaining](./durable-task-sdks/dotnet/FunctionChaining) | Sequential workflow with data transformation |
| Python | [Function Chaining](./durable-task-sdks/python/function-chaining) | Sequential workflow basics |
| Java | [Function Chaining](./durable-task-sdks/java/function-chaining) | Sequential workflow basics |

---

## 📋 Samples by Pattern

A quick-reference matrix showing which patterns are available in each language and framework.

### Durable Task SDKs

| Pattern | .NET | Python | Java |
|---------|------|--------|------|
| Function Chaining | [✅](./durable-task-sdks/dotnet/FunctionChaining) | [✅](./durable-task-sdks/python/function-chaining) | [✅](./durable-task-sdks/java/function-chaining) |
| Fan-out/Fan-in | [✅](./durable-task-sdks/dotnet/FanOutFanIn) | [✅](./durable-task-sdks/python/fan-out-fan-in) | [✅](./durable-task-sdks/java/fan-out-fan-in) |
| Human Interaction | [✅](./durable-task-sdks/dotnet/HumanInteraction) | [✅](./durable-task-sdks/python/human-interaction) | [✅](./durable-task-sdks/java/human-interaction) |
| Async HTTP API | | [✅](./durable-task-sdks/python/async-http-api) | [✅](./durable-task-sdks/java/async-http-api) |
| Monitoring | [✅](./durable-task-sdks/dotnet/Monitoring) | [✅](./durable-task-sdks/python/monitoring) | [✅](./durable-task-sdks/java/monitoring) |
| Sub-orchestrations | [✅](./durable-task-sdks/dotnet/SubOrchestrations) | [✅](./durable-task-sdks/python/sub-orchestrations) | [✅](./durable-task-sdks/java/sub-orchestrations) |
| Eternal Orchestrations | [✅](./durable-task-sdks/dotnet/EternalOrchestrations) | [✅](./durable-task-sdks/python/eternal-orchestrations) | [✅](./durable-task-sdks/java/eternal-orchestrations) |
| Saga Pattern | | [✅](./durable-task-sdks/python/saga) | |
| Durable Entities | [✅](./durable-task-sdks/dotnet/EntitiesSample) | [✅](./durable-task-sdks/python/entities) | |
| Orchestration Versioning | [✅](./durable-task-sdks/dotnet/OrchestrationVersioning) | [✅](./durable-task-sdks/python/versioning) | |
| ASP.NET Web API | [✅](./durable-task-sdks/dotnet/AspNetWebApp) | | |
| Scheduled Tasks | [✅](./durable-task-sdks/dotnet/ScheduleWebApp) | | |
| .NET Aspire Integration | [✅](./durable-task-sdks/dotnet/DtsWithAspire) | | |
| AI Agent Chaining | [✅](./durable-task-sdks/dotnet/Agents/PromptChaining) | | |
| AI Research Agent | | [✅](./durable-task-sdks/python/arXiv_research_agent) | |

### Durable Functions

| Pattern | .NET | Python | Java | JavaScript |
|---------|------|--------|------|------------|
| Hello Cities (Quickstart) | [✅](./durable-functions/dotnet/HelloCities) | | [✅](./durable-functions/java/HelloCities) | [✅](./durable-functions/javascript/HelloCities) |
| Fan-out/Fan-in | | [✅](./durable-functions/python/fan-out-fan-in) | [✅](./durable-functions/java/HelloCities) | [✅](./durable-functions/javascript/HelloCities) |
| Order Processor | [✅](./durable-functions/dotnet/OrderProcessor) | | | |
| Saga Pattern | [✅](./durable-functions/dotnet/Saga) | | | |
| Distributed Tracing | [✅](./durable-functions/dotnet/DistributedTracing) | | | |
| PDF Summarizer | [✅](./durable-functions/dotnet/PdfSummarizer) | [✅](./durable-functions/python/pdf-summarizer) | | |
| AI Travel Planner | [✅](./durable-functions/dotnet/AiAgentTravelPlanOrchestrator) | | | |
| Aspire Integration | [✅](./durable-functions/dotnet/AzureFunctionsAndDtsWithAspire) | | | |

---

## Durable Task SDKs

### .NET

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Function Chaining](./durable-task-sdks/dotnet/FunctionChaining) | Function Chaining | Sequential workflow with data transformation |
| [Fan-out/Fan-in](./durable-task-sdks/dotnet/FanOutFanIn) | Fan-out/Fan-in | Parallel execution and result aggregation |
| [Human Interaction](./durable-task-sdks/dotnet/HumanInteraction) | Human Interaction | Approval workflow with external events and timeouts |
| [Durable Entities](./durable-task-sdks/dotnet/EntitiesSample) | Durable Entities | Funds transfer using stateful distributed objects |
| [Orchestration Versioning](./durable-task-sdks/dotnet/OrchestrationVersioning) | Versioning | Safe evolution of running orchestrations |
| [ASP.NET Web API](./durable-task-sdks/dotnet/AspNetWebApp) | Web API | Web API running orchestrations |
| [Scheduled Tasks](./durable-task-sdks/dotnet/ScheduleWebApp) | Scheduled Tasks | Recurring background tasks with scheduled orchestrations |
| [.NET Aspire Integration](./durable-task-sdks/dotnet/DtsWithAspire) | Aspire | Local dev orchestration with Aspire |
| [AI Agent Chaining](./durable-task-sdks/dotnet/Agents/PromptChaining) | AI Agents | Multi-agent workflow with research, content, and image agents |
| [Monitoring](./durable-task-sdks/dotnet/Monitoring) | Monitoring | Periodic polling pattern with ContinueAsNew |
| [Sub-Orchestrations](./durable-task-sdks/dotnet/SubOrchestrations) | Sub-orchestrations | Parent/child orchestration composition for order processing |
| [Eternal Orchestrations](./durable-task-sdks/dotnet/EternalOrchestrations) | Eternal Orchestrations | Indefinitely running orchestration with ContinueAsNew |
| [OpenTelemetry Tracing](./durable-task-sdks/dotnet/OpenTelemetryTracing) | Observability | Distributed tracing with OpenTelemetry and Jaeger |

### Python

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Function Chaining](./durable-task-sdks/python/function-chaining) | Function Chaining | Sequential workflow basics |
| [Fan-out/Fan-in](./durable-task-sdks/python/fan-out-fan-in) | Fan-out/Fan-in | Parallel execution and result aggregation |
| [Human Interaction](./durable-task-sdks/python/human-interaction) | Human Interaction | Approval workflow with external events and timeouts |
| [Async HTTP API](./durable-task-sdks/python/async-http-api) | Async HTTP API | FastAPI with long-running operations |
| [Monitoring](./durable-task-sdks/python/monitoring) | Monitoring | Periodic polling pattern |
| [Sub-orchestrations](./durable-task-sdks/python/sub-orchestrations) | Sub-orchestrations | Nested orchestration composition |
| [Eternal Orchestrations](./durable-task-sdks/python/eternal-orchestrations) | Eternal Orchestrations | Continue-as-new pattern |
| [Durable Entities](./durable-task-sdks/python/entities) | Durable Entities | Counter entity |
| [Orchestration Versioning](./durable-task-sdks/python/versioning) | Versioning | Safe evolution of running orchestrations |
| [AI Research Agent](./durable-task-sdks/python/arXiv_research_agent) | AI Agents | Autonomous research agent with arXiv + LLM |
| [Saga Pattern](./durable-task-sdks/python/saga) | Saga | Travel booking with compensating transactions |
| [OpenTelemetry Tracing](./durable-task-sdks/python/opentelemetry-tracing) | Observability | Distributed tracing with OpenTelemetry and Jaeger |

### Java

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Function Chaining](./durable-task-sdks/java/function-chaining) | Function Chaining | Sequential workflow basics |
| [Fan-out/Fan-in](./durable-task-sdks/java/fan-out-fan-in) | Fan-out/Fan-in | Parallel execution and result aggregation |
| [Human Interaction](./durable-task-sdks/java/human-interaction) | Human Interaction | Approval workflow with external events and timeouts |
| [Async HTTP API](./durable-task-sdks/java/async-http-api) | Async HTTP API | Long-running operations with HTTP polling |
| [Monitoring](./durable-task-sdks/java/monitoring) | Monitoring | Periodic polling pattern |
| [Sub-orchestrations](./durable-task-sdks/java/sub-orchestrations) | Sub-orchestrations | Nested orchestration composition |
| [Eternal Orchestrations](./durable-task-sdks/java/eternal-orchestrations) | Eternal Orchestrations | Continue-as-new pattern |

---

## Durable Functions

### .NET

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Hello Cities](./durable-functions/dotnet/HelloCities) | Quickstart | Basic orchestration with 3 activities |
| [Order Processor](./durable-functions/dotnet/OrderProcessor) | Order Workflow | End-to-end order processing workflow |
| [Saga Pattern](./durable-functions/dotnet/Saga) | Saga | Compensating transactions for distributed operations |
| [Aspire Integration](./durable-functions/dotnet/AzureFunctionsAndDtsWithAspire) | Aspire | Azure Functions + DTS with Aspire |
| [PDF Summarizer](./durable-functions/dotnet/PdfSummarizer) | AI Pipeline | AI-powered document processing pipeline |
| [AI Travel Planner](./durable-functions/dotnet/AiAgentTravelPlanOrchestrator) | AI Agents | Multi-agent travel planning orchestration |
| [Distributed Tracing](./durable-functions/dotnet/DistributedTracing) | Observability | Distributed tracing with Application Insights and Jaeger |

### Python

| Sample | Pattern | Description |
|--------|---------|-------------|
| [PDF Summarizer](./durable-functions/python/pdf-summarizer) | AI Pipeline | AI-powered PDF summarization |
| [Fan-out/Fan-in](./durable-functions/python/fan-out-fan-in) | Fan-out/Fan-in | Parallel processing with result aggregation |

### Java

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Hello Cities](./durable-functions/java/HelloCities) | Function Chaining, Fan-out/Fan-in | Java quickstart with sequential and parallel orchestration patterns |

### JavaScript

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Hello Cities](./durable-functions/javascript/HelloCities) | Function Chaining, Fan-out/Fan-in | JavaScript quickstart with sequential and parallel orchestration patterns |

---

## Scenarios

| Sample | Description |
|--------|-------------|
| [Autoscaling in ACA](./scenarios/AutoscalingInACA) | KEDA-based dynamic worker scaling in Azure Container Apps |

---

## Contributing a Sample

Want to add your own sample? See the [Contributing Guide](../CONTRIBUTING.md) for guidelines on sample structure, documentation, and submission.
