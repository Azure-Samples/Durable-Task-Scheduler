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
| JavaScript | [Function Chaining](./durable-task-sdks/javascript/function-chaining) | Sequential workflow basics |
| Go | [Function Chaining](./durable-task-sdks/go/function-chaining) | Worker and client in one runnable package (Go 1.25+, beta SDK) |

---

## 📋 Samples by Pattern

A quick-reference matrix showing which patterns are available in each language and framework.

### Durable Task SDKs

| Pattern | .NET | Python | Java | JavaScript | Go (beta) |
|---------|------|--------|------|------------|-----------|
| Function Chaining | [✅](./durable-task-sdks/dotnet/FunctionChaining) | [✅](./durable-task-sdks/python/function-chaining) | [✅](./durable-task-sdks/java/function-chaining) | [✅](./durable-task-sdks/javascript/function-chaining) | [✅](./durable-task-sdks/go/function-chaining) |
| Fan-out/Fan-in | [✅](./durable-task-sdks/dotnet/FanOutFanIn) | [✅](./durable-task-sdks/python/fan-out-fan-in) | [✅](./durable-task-sdks/java/fan-out-fan-in) | [✅](./durable-task-sdks/javascript/fan-out-fan-in) | [✅](./durable-task-sdks/go/fan-out-fan-in) |
| Human Interaction | [✅](./durable-task-sdks/dotnet/HumanInteraction) | [✅](./durable-task-sdks/python/human-interaction) | [✅](./durable-task-sdks/java/human-interaction) | | [✅](./durable-task-sdks/go/human-interaction) |
| Async HTTP API | | [✅](./durable-task-sdks/python/async-http-api) | [✅](./durable-task-sdks/java/async-http-api) | | [✅](./durable-task-sdks/go/async-http-api) |
| Monitoring | [✅](./durable-task-sdks/dotnet/Monitoring) | [✅](./durable-task-sdks/python/monitoring) | [✅](./durable-task-sdks/java/monitoring) | | [✅](./durable-task-sdks/go/monitoring) |
| Sub-orchestrations | [✅](./durable-task-sdks/dotnet/SubOrchestrations) | [✅](./durable-task-sdks/python/sub-orchestrations) | [✅](./durable-task-sdks/java/sub-orchestrations) | | [✅](./durable-task-sdks/go/sub-orchestrations) |
| Eternal Orchestrations | [✅](./durable-task-sdks/dotnet/EternalOrchestrations) | [✅](./durable-task-sdks/python/eternal-orchestrations) | [✅](./durable-task-sdks/java/eternal-orchestrations) | | [✅](./durable-task-sdks/go/eternal-orchestrations) |
| Saga Pattern | | [✅](./durable-task-sdks/python/saga) | | | [✅](./durable-task-sdks/go/saga) |
| Durable Entities | [✅](./durable-task-sdks/dotnet/EntitiesSample) | [✅](./durable-task-sdks/python/entities) | | | [✅](./durable-task-sdks/go/entities) |
| Orchestration Versioning | [✅](./durable-task-sdks/dotnet/OrchestrationVersioning) | [✅](./durable-task-sdks/python/versioning) | | | [✅](./durable-task-sdks/go/versioning) |
| ASP.NET Web API | [✅](./durable-task-sdks/dotnet/AspNetWebApp) | | | | |
| Scheduled Tasks | [✅](./durable-task-sdks/dotnet/ScheduleWebApp) | [✅](./durable-task-sdks/python/scheduled-tasks) | | | [✅](./durable-task-sdks/go/scheduled-tasks) |
| .NET Aspire Integration | [✅](./durable-task-sdks/dotnet/DtsWithAspire) | | | | |
| AI Agent Chaining | [✅](./durable-task-sdks/dotnet/Agents/PromptChaining) | | | | |
| AI Research Agent | | [✅](./durable-task-sdks/python/arXiv_research_agent) | | | [✅](./durable-task-sdks/go/arXiv_research_agent) |
| Agent-Directed Workflows | [✅](./durable-task-sdks/dotnet/Agents/AgentDirectedWorkflows) | [✅](./durable-task-sdks/python/agent-directed-workflows) | | | [✅](./durable-task-sdks/go/agent-directed-workflows) |
| Large Payload | [✅](./durable-task-sdks/dotnet/LargePayload) | [✅](./durable-task-sdks/python/large-payload) | | | [✅](./durable-task-sdks/go/large-payload) |
| Export History | [✅](./durable-task-sdks/dotnet/ExportHistoryWebApp) | [✅](./durable-task-sdks/python/history-export) | | | [✅](./durable-task-sdks/go/history-export) |
| Bounded Coordinator | [✅](./durable-task-sdks/dotnet/BoundedCoordinator) | [✅](./durable-task-sdks/python/bounded-coordinator) | | | [✅](./durable-task-sdks/go/bounded-coordinator) |
| OpenTelemetry Tracing | [✅](./durable-task-sdks/dotnet/OpenTelemetryTracing) | [✅](./durable-task-sdks/python/opentelemetry-tracing) | [✅](./durable-task-sdks/java/opentelemetry-tracing) | | [✅](./durable-task-sdks/go/opentelemetry-tracing) |
| Orchestration Management | | [✅](./durable-task-sdks/python/orchestration-management) | | | [✅](./durable-task-sdks/go/orchestration-management) |
| Testing | | [✅](./durable-task-sdks/python/testing) | | | [✅](./durable-task-sdks/go/testing) |
| Work Item Filtering | | [✅](./durable-task-sdks/python/work-item-filtering) | | | [✅](./durable-task-sdks/go/work-item-filtering) |

The [Go samples](./durable-task-sdks/go/) cover self-hosted workflows, durable entities, scheduling, and integrations using Durable Task Scheduler.

### Durable Functions

| Pattern | .NET | Python | Java | JavaScript | PowerShell |
|---------|------|--------|------|------------|------------|
| Hello Cities (Quickstart) | [✅](./durable-functions/dotnet/HelloCities) | | [✅](./durable-functions/java/HelloCities) | [✅](./durable-functions/javascript/HelloCities) | [✅](./durable-functions/powershell/HelloCities) |
| Fan-out/Fan-in | | [✅](./durable-functions/python/fan-out-fan-in) | [✅](./durable-functions/java/HelloCities) | [✅](./durable-functions/javascript/HelloCities) | [✅](./durable-functions/powershell/HelloCities) |
| Order Processor | [✅](./durable-functions/dotnet/OrderProcessor) | | | |
| Saga Pattern | [✅](./durable-functions/dotnet/Saga) | | | |
| Distributed Tracing | [✅](./durable-functions/dotnet/DistributedTracing) | | | |
| Large Payload | [✅](./durable-functions/dotnet/LargePayload) | | | |
| Large Payload Fan-out/Fan-in | [✅](./durable-functions/dotnet/LargePayloadFanOutFanIn) | | | |
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
| [Large Payload](./durable-task-sdks/dotnet/LargePayload) | Large Payload | Blob-backed payload externalization for orchestration inputs and outputs larger than 1 MB |

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
| [Agent-Directed Workflows](./durable-task-sdks/python/agent-directed-workflows) | AI Agents | Entity-backed agent loop with durable tool calls |
| [Bounded Coordinator](./durable-task-sdks/python/bounded-coordinator) | Bounded Coordinator | Bounded child batches with continue-as-new |
| [Large Payload](./durable-task-sdks/python/large-payload) | Large Payload | Externalize large payloads to Azure Blob Storage |
| [Export History](./durable-task-sdks/python/history-export) | History Export | Export terminal orchestration histories to Azure Blob Storage |
| [Scheduled Tasks](./durable-task-sdks/python/scheduled-tasks) | Scheduled Tasks | Recurring interval schedules and management |
| [Orchestration Management](./durable-task-sdks/python/orchestration-management) | Management | Query, restart, and purge orchestration instances |
| [Testing](./durable-task-sdks/python/testing) | Testing | Test orchestrations and activities |
| [Work Item Filtering](./durable-task-sdks/python/work-item-filtering) | Worker Routing | Filter work by registered task names and versions |

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

### JavaScript

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Function Chaining](./durable-task-sdks/javascript/function-chaining) | Function Chaining | Sequential workflow basics with JavaScript SDK |
| [Fan-out/Fan-in](./durable-task-sdks/javascript/fan-out-fan-in) | Fan-out/Fan-in | Parallel execution and result aggregation with JavaScript SDK |

### Go

**Requires:** Go **1.25.0+**, using `github.com/microsoft/durabletask-go` **v1.0.0-beta.1**. All 20 packages share the module in [`durable-task-sdks/go`](./durable-task-sdks/go).

From the repository root, with the emulator running:

```bash
cd samples/durable-task-sdks/go
go mod download
go run ./function-chaining
```

Substitute any directory name below in `go run ./<name>`. By default, each sample starts its worker and client together, checks its result, and exits; HTTP/agent samples also document optional interactive modes. The default connection string is `Endpoint=http://localhost:8080;TaskHub=default;Authentication=None`; override it with `DTS_CONNECTION_STRING` to connect to Azure. See the [Go quickstart](../docs/quickstart.md#go) and each README for feature-specific configuration. These are runnable SDK examples, not `azd` deployment templates.

Large Payload and Export History also require blob storage and default to local Azurite. Changing `DTS_CONNECTION_STRING` does not switch storage to Azure Blob Storage. History-export requires an isolated emulator or Azure task hub with no other export workers and no unrelated workloads completing during its export window; both emulator and Azure runs require `HISTORY_EXPORT_ISOLATED_TASKHUB=1` to confirm isolation. The flag does not create or isolate a hub. Review the sample READMEs before running.

The AI demonstrations use explicit echo/synthetic fixtures by default on both backends; real arXiv or model calls require separate configuration. Scheduled tasks use Go-owned schedule state, not a shared cross-SDK schedule contract.

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Function Chaining](./durable-task-sdks/go/function-chaining) | Function Chaining | Sequential activities with result verification |
| [Fan-out/Fan-in](./durable-task-sdks/go/fan-out-fan-in) | Fan-out/Fan-in | Parallel activities and result aggregation |
| [Human Interaction](./durable-task-sdks/go/human-interaction) | Human Interaction | External events with an approval timeout |
| [Async HTTP API](./durable-task-sdks/go/async-http-api) | Async HTTP API | HTTP start/status endpoints and a polling client |
| [Monitoring](./durable-task-sdks/go/monitoring) | Monitoring | Periodic status checks with durable timers |
| [Sub-orchestrations](./durable-task-sdks/go/sub-orchestrations) | Sub-orchestrations | Parent/child workflow composition |
| [Eternal Orchestrations](./durable-task-sdks/go/eternal-orchestrations) | Eternal Orchestrations | Continue-as-new with a bounded demonstration run |
| [Durable Entities](./durable-task-sdks/go/entities) | Durable Entities | Persistent entity state and operations |
| [Orchestration Versioning](./durable-task-sdks/go/versioning) | Versioning | Register and run versioned tasks |
| [AI Research Agent](./durable-task-sdks/go/arXiv_research_agent) | AI Agents | Durable research pipeline with synthetic fixtures and optional arXiv/Azure OpenAI mode |
| [Saga Pattern](./durable-task-sdks/go/saga) | Saga | Compensating activities after a failed step |
| [OpenTelemetry Tracing](./durable-task-sdks/go/opentelemetry-tracing) | Observability | Application spans and W3C trace-context propagation; durable spans are DTS-owned |
| [Agent-Directed Workflows](./durable-task-sdks/go/agent-directed-workflows) | AI Agents | Durable entity conversations with HTTP/SSE; echo mode by default |
| [Bounded Coordinator](./durable-task-sdks/go/bounded-coordinator) | Bounded Coordinator | Bounded child batches with continue-as-new |
| [Large Payload](./durable-task-sdks/go/large-payload) | Large Payload | Externalize inputs and outputs with the payload extension |
| [Export History](./durable-task-sdks/go/history-export) | History Export (preview) | Export terminal orchestration histories with the history extension |
| [Scheduled Tasks](./durable-task-sdks/go/scheduled-tasks) | Scheduled Tasks | Recurring interval schedules and lifecycle management |
| [Orchestration Management](./durable-task-sdks/go/orchestration-management) | Management | Query and manage orchestration lifecycle |
| [Testing](./durable-task-sdks/go/testing) | Testing | Local step-adapter tests and opt-in DTS replay/integration checks |
| [Work Item Filtering](./durable-task-sdks/go/work-item-filtering) | Worker Routing | Route work to matching worker registrations |

Run `go build ./...`, `go test ./...`, and `go vet ./...` from the Go module without starting a scheduler. Scheduler-backed tests require explicit `DTS_SAMPLES_E2E=1` opt-in; see [contributor validation instructions](../CONTRIBUTING.md#go-samples).

The Go testing sample exercises a shared order workflow offline through a local step adapter. This is not an SDK in-memory backend, and those unit tests do not validate Durable Task replay. Replay and SDK integration require the opt-in tests against a real DTS emulator or Azure scheduler.

---

## Durable Functions

### .NET

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Hello Cities](./durable-functions/dotnet/HelloCities) | Quickstart | Basic orchestration with 3 activities |
| [Order Processor](./durable-functions/dotnet/OrderProcessor) | Order Workflow | End-to-end order processing workflow |
| [Saga Pattern](./durable-functions/dotnet/Saga) | Saga | Compensating transactions for distributed operations |
| [Aspire Integration](./durable-functions/dotnet/AzureFunctionsAndDtsWithAspire) | Aspire | Azure Functions + DTS with Aspire |
| [Large Payload](./durable-functions/dotnet/LargePayload) | Large Payload | Single round-trip orchestration that externalizes payloads larger than 1 MB to blob storage |
| [Large Payload Fan-out/Fan-in](./durable-functions/dotnet/LargePayloadFanOutFanIn) | Large Payload | Parallel orchestration that externalizes payloads larger than 1 MB for each activity result |
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

### PowerShell

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Hello Cities](./durable-functions/powershell/HelloCities) | Function Chaining, Fan-out/Fan-in | PowerShell quickstart with sequential and parallel orchestration patterns |

---

## Durable Extension for Microsoft Agent Framework

The [Durable Task extension for Microsoft Agent Framework](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-microsoft-agent-framework) lets you make any [Microsoft Agent Framework](https://github.com/microsoft/agent-framework) agent durable with persistent sessions, built-in API endpoints, and distributed scaling — without changes to your agent logic. It also supports graph-based workflows via `WorkflowBuilder`.

> **Setup:** These samples live in the `microsoft/agent-framework` repo and are included here via a Git submodule. Run `git submodule update --init external/agent-framework` after cloning.

📂 **[Full details and sample index →](./durable-extension-for-agent-framework/)**

### Durable Agents (.NET)

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Single Agent](./durable-extension-for-agent-framework/dotnet/hosting/azure-functions/01-SingleAgent) | Azure Functions | Basic durable agent with persistent sessions |
| [Orchestration Chaining](./durable-extension-for-agent-framework/dotnet/hosting/azure-functions/02-OrchestrationChaining) | Azure Functions | Multi-agent sequential orchestration |
| [Orchestration Concurrency](./durable-extension-for-agent-framework/dotnet/hosting/azure-functions/03-OrchestrationConcurrency) | Azure Functions | Parallel agent execution (fan-out/fan-in) |
| [Orchestration Conditionals](./durable-extension-for-agent-framework/dotnet/hosting/azure-functions/04-OrchestrationConditionals) | Azure Functions | Conditional routing between agents |
| [Human-in-the-Loop](./durable-extension-for-agent-framework/dotnet/hosting/azure-functions/05-HumanInTheLoop) | Azure Functions | Agent pauses for human approval |
| [Reliable Streaming](./durable-extension-for-agent-framework/dotnet/hosting/azure-functions/06-ReliableStreaming) | Azure Functions | Real-time token streaming with durability |
| [Single Agent](./durable-extension-for-agent-framework/dotnet/hosting/console-apps/01-SingleAgent) | Console App | Same pattern without Azure Functions |

### Durable MAF Workflows (.NET)

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Sequential](./durable-extension-for-agent-framework/dotnet/durable-maf-workflows/azure-functions/01-Sequential) | Azure Functions | Order cancellation pipeline: look up → cancel → notify |
| [Concurrent](./durable-extension-for-agent-framework/dotnet/durable-maf-workflows/azure-functions/02-Concurrent) | Azure Functions | Fan-out to multiple expert agents, fan-in to aggregate |
| [Human-in-the-Loop](./durable-extension-for-agent-framework/dotnet/durable-maf-workflows/azure-functions/03-HumanInTheLoop) | Azure Functions | Expense reimbursement with multi-stage approvals |
| [Sequential](./durable-extension-for-agent-framework/dotnet/durable-maf-workflows/console-apps/01-Sequential) | Console App | Sequential executor pipeline |
| [Conditional Edges](./durable-extension-for-agent-framework/dotnet/durable-maf-workflows/console-apps/03-ConditionalEdges) | Console App | Runtime routing based on conditions |

### Durable Agents (Python)

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Single Agent](./durable-extension-for-agent-framework/python/hosting/azure-functions/01-single-agent) | Azure Functions | Basic durable agent with persistent sessions |
| [Multi-Agent Orchestration](./durable-extension-for-agent-framework/python/hosting/azure-functions/02-multi-agent-orchestration) | Azure Functions | Multiple agents in a durable orchestration |
| [Tool Calling](./durable-extension-for-agent-framework/python/hosting/azure-functions/03-tool-calling) | Azure Functions | Agent with function tools |
| [Single Agent](./durable-extension-for-agent-framework/python/hosting/durable-task/01-single-agent) | Durable Task SDK | Agent hosted with DT SDK directly |

---

## Scenarios

| Sample | Description |
|--------|-------------|
| [Autoscaling in ACA](./scenarios/AutoscalingInACA) | KEDA-based dynamic worker scaling in Azure Container Apps |

---

## Contributing a Sample

Want to add your own sample? See the [Contributing Guide](../CONTRIBUTING.md) for guidelines on sample structure, documentation, and submission.
