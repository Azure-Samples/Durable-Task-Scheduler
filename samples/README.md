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

---

## 📋 Samples by Pattern

A quick-reference matrix showing which patterns are available in each language and framework.

### Durable Task SDKs

| Pattern | .NET | Python | Java | JavaScript |
|---------|------|--------|------|------------|
| Function Chaining | [✅](./durable-task-sdks/dotnet/FunctionChaining) | [✅](./durable-task-sdks/python/function-chaining) | [✅](./durable-task-sdks/java/function-chaining) | [✅](./durable-task-sdks/javascript/function-chaining) |
| Fan-out/Fan-in | [✅](./durable-task-sdks/dotnet/FanOutFanIn) | [✅](./durable-task-sdks/python/fan-out-fan-in) | [✅](./durable-task-sdks/java/fan-out-fan-in) | [✅](./durable-task-sdks/javascript/fan-out-fan-in) |
| Human Interaction | [✅](./durable-task-sdks/dotnet/HumanInteraction) | [✅](./durable-task-sdks/python/human-interaction) | [✅](./durable-task-sdks/java/human-interaction) | |
| Async HTTP API | | [✅](./durable-task-sdks/python/async-http-api) | [✅](./durable-task-sdks/java/async-http-api) | |
| Monitoring | [✅](./durable-task-sdks/dotnet/Monitoring) | [✅](./durable-task-sdks/python/monitoring) | [✅](./durable-task-sdks/java/monitoring) | |
| Sub-orchestrations | [✅](./durable-task-sdks/dotnet/SubOrchestrations) | [✅](./durable-task-sdks/python/sub-orchestrations) | [✅](./durable-task-sdks/java/sub-orchestrations) | |
| Eternal Orchestrations | [✅](./durable-task-sdks/dotnet/EternalOrchestrations) | [✅](./durable-task-sdks/python/eternal-orchestrations) | [✅](./durable-task-sdks/java/eternal-orchestrations) | |
| Saga Pattern | | [✅](./durable-task-sdks/python/saga) | | |
| Durable Entities | [✅](./durable-task-sdks/dotnet/EntitiesSample) | [✅](./durable-task-sdks/python/entities) | [✅](./durable-task-sdks/java/entities) | |
| Orchestration Versioning | [✅](./durable-task-sdks/dotnet/OrchestrationVersioning) | [✅](./durable-task-sdks/python/versioning) | | |
| ASP.NET Web API | [✅](./durable-task-sdks/dotnet/AspNetWebApp) | | | |
| Scheduled Tasks | [✅](./durable-task-sdks/dotnet/ScheduleWebApp) | [✅](./durable-task-sdks/python/scheduled-tasks) | | |
| .NET Aspire Integration | [✅](./durable-task-sdks/dotnet/DtsWithAspire) | | | |
| AI Agent Chaining | [✅](./durable-task-sdks/dotnet/Agents/PromptChaining) | | | |
| AI Research Agent | | [✅](./durable-task-sdks/python/arXiv_research_agent) | | |
| Large Payload | [✅](./durable-task-sdks/dotnet/LargePayload) | [✅](./durable-task-sdks/python/large-payload) | | |
| Export History | [✅](./durable-task-sdks/dotnet/ExportHistoryWebApp) | [✅](./durable-task-sdks/python/history-export) | | |
| Bounded Coordinator | [✅](./durable-task-sdks/dotnet/BoundedCoordinator) | [✅](./durable-task-sdks/python/bounded-coordinator) | | |
| OpenTelemetry Tracing | [✅](./durable-task-sdks/dotnet/OpenTelemetryTracing) | [✅](./durable-task-sdks/python/opentelemetry-tracing) | [✅](./durable-task-sdks/java/opentelemetry-tracing) | |
| Agent-Directed Workflows | [✅](./durable-task-sdks/dotnet/Agents/AgentDirectedWorkflows) | [✅](./durable-task-sdks/python/agent-directed-workflows) | | |
| Orchestration Management | | [✅](./durable-task-sdks/python/orchestration-management) | | |
| Work Item Filtering | [✅](./scenarios/WorkItemFilteringSplitActivities) | [✅](./durable-task-sdks/python/work-item-filtering) | [✅](./scenarios/WorkItemFilteringSplitActivitiesJava) | |
| In-Memory Testing | | [✅](./durable-task-sdks/python/testing) | | |

### Durable Functions

| Pattern | .NET | Python | Java | JavaScript | TypeScript | PowerShell |
|---------|------|--------|------|------------|------------|------------|
| Hello Cities (Quickstart) | [✅](./durable-functions/dotnet/HelloCities) | [✅](./durable-functions/python/hello-cities) | [✅](./durable-functions/java/HelloCities) | [✅](./durable-functions/javascript/HelloCities) | [✅](./durable-functions/typescript/HelloCities) | [✅](./durable-functions/powershell/HelloCities) |
| Fan-out/Fan-in | [✅](./durable-functions/dotnet/LargePayloadFanOutFanIn) | [✅](./durable-functions/python/fan-out-fan-in) | [✅](./durable-functions/java/HelloCities) | [✅](./durable-functions/javascript/HelloCities) | [✅](./durable-functions/typescript/HelloCities) | [✅](./durable-functions/powershell/HelloCities) |
| Order Processor | [✅](./durable-functions/dotnet/OrderProcessor) | | | | | |
| Saga Pattern | [✅](./durable-functions/dotnet/Saga) | | | | | |
| Distributed Tracing | [✅](./durable-functions/dotnet/DistributedTracing) | | | | | |
| Large Payload | [✅](./durable-functions/dotnet/LargePayload) | | | | | |
| Large Payload Fan-out/Fan-in | [✅](./durable-functions/dotnet/LargePayloadFanOutFanIn) | | | | | |
| AI Travel Planner | [✅](./durable-functions/dotnet/AiAgentTravelPlanOrchestrator) | | | | | |
| Aspire Integration | [✅](./durable-functions/dotnet/AzureFunctionsAndDtsWithAspire) | | | | | |
| Agent-Directed Workflows | [✅](./durable-functions/dotnet/AgentDirectedWorkflows) | [✅](./durable-functions/python/agent-directed-workflows) | | | | |
| Durable Entities | | | [✅](./durable-functions/java/Entities) | | | |
| Work Item Filtering | [✅](./durable-functions/dotnet/WorkItemFiltering) | [✅](./durable-functions/python/work-item-filtering) | | | | |
| Custom Exception Properties | [✅](./durable-functions/dotnet/CustomExceptionPropertyProvider) | | | [✅](./durable-functions/javascript/CustomExceptionPropertyProvider) | | |
| Critical Sections | | | | [✅](./durable-functions/javascript/CriticalSections) | [✅](./durable-functions/typescript/CriticalSections) | |

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
| [Export History](./durable-task-sdks/dotnet/ExportHistoryWebApp) | Export History | REST API for batch and continuous orchestration history export to Azure Blob Storage |
| [Bounded Coordinator](./durable-task-sdks/dotnet/BoundedCoordinator) | Bounded Coordinator | Process child work in bounded batches and reset history with ContinueAsNew |
| [Agent-Directed Workflows](./durable-task-sdks/dotnet/Agents/AgentDirectedWorkflows) | AI Agents | Entity-based agent loop with tool calling and Redis-backed response streaming |

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
| [Scheduled Tasks](./durable-task-sdks/python/scheduled-tasks) | Scheduled Tasks | Recurring orchestrations with pause, resume, and schedule management |
| [Export History](./durable-task-sdks/python/history-export) | Export History | Export orchestration history to Azure Blob Storage |
| [Bounded Coordinator](./durable-task-sdks/python/bounded-coordinator) | Bounded Coordinator | Process child work in bounded batches and reset history with continue-as-new |
| [Large Payload](./durable-task-sdks/python/large-payload) | Large Payload | Transparently externalize large payloads to Azure Blob Storage |
| [Orchestration Management](./durable-task-sdks/python/orchestration-management) | Management | Query, suspend, resume, terminate, and purge orchestration instances |
| [Work Item Filtering](./durable-task-sdks/python/work-item-filtering) | Work Item Filtering | Route orchestration and activity work to workers that register matching tasks |
| [In-Memory Testing](./durable-task-sdks/python/testing) | Testing | Test orchestrations and activities without Docker, an emulator, or Azure resources |
| [Agent-Directed Workflows](./durable-task-sdks/python/agent-directed-workflows) | AI Agents | Entity-based agent loop with tool calling and response streaming |

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
| [Durable Entities](./durable-task-sdks/java/entities) | Durable Entities | Counter entity with state persisted across operations |
| [OpenTelemetry Tracing](./durable-task-sdks/java/opentelemetry-tracing) | Observability | Distributed tracing with OpenTelemetry and Jaeger |

### JavaScript / TypeScript (preview)

The [JavaScript/TypeScript SDK](https://github.com/microsoft/durabletask-js) is available in preview. The samples below use JavaScript; see the [JavaScript sample guide](./durable-task-sdks/javascript/) for prerequisites and setup.

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Function Chaining](./durable-task-sdks/javascript/function-chaining) | Function Chaining | Sequential workflow basics with JavaScript SDK |
| [Fan-out/Fan-in](./durable-task-sdks/javascript/fan-out-fan-in) | Fan-out/Fan-in | Parallel execution and result aggregation with JavaScript SDK |

### Go (samples coming soon)

Go samples in this repository are coming soon. The [official Go SDK repository](https://github.com/microsoft/durabletask-go) provides the beta SDK and its own samples for Durable Task Scheduler and the local emulator.

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
| [AI Travel Planner](./durable-functions/dotnet/AiAgentTravelPlanOrchestrator) | AI Agents | Multi-agent travel planning orchestration |
| [Distributed Tracing](./durable-functions/dotnet/DistributedTracing) | Observability | Distributed tracing with Application Insights and Jaeger |
| [Agent-Directed Workflows](./durable-functions/dotnet/AgentDirectedWorkflows) | AI Agents | Entity-based agent loop with tool calling and Redis-backed response streaming |
| [Work Item Filtering](./durable-functions/dotnet/WorkItemFiltering) | Work Item Filtering | Route work between two Function apps sharing a task hub |
| [Custom Exception Properties](./durable-functions/dotnet/CustomExceptionPropertyProvider) | Error Handling | Propagate structured exception properties to orchestration failure details |

### Python

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Hello Cities](./durable-functions/python/hello-cities) | Quickstart | Basic orchestration with greeting activities |
| [Fan-out/Fan-in](./durable-functions/python/fan-out-fan-in) | Fan-out/Fan-in | Parallel processing with result aggregation |
| [Agent-Directed Workflows](./durable-functions/python/agent-directed-workflows) | AI Agents | Entity-based agent loop with tool calling and response streaming |
| [Work Item Filtering](./durable-functions/python/work-item-filtering) | Work Item Filtering | Route work between two Function apps sharing a task hub |

### Java

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Hello Cities](./durable-functions/java/HelloCities) | Function Chaining, Fan-out/Fan-in | Java quickstart with sequential and parallel orchestration patterns |
| [Durable Entities](./durable-functions/java/Entities) | Durable Entities | Stateful counter entity |

### JavaScript

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Hello Cities](./durable-functions/javascript/HelloCities) | Function Chaining, Fan-out/Fan-in | JavaScript quickstart with sequential and parallel orchestration patterns |
| [Critical Sections](./durable-functions/javascript/CriticalSections) | Entity Coordination | Coordinate access to entities with durable locks |
| [Custom Exception Properties](./durable-functions/javascript/CustomExceptionPropertyProvider) | Error Handling | Propagate structured exception properties to orchestration failure details |

### TypeScript

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Hello Cities](./durable-functions/typescript/HelloCities) | Function Chaining, Fan-out/Fan-in | TypeScript quickstart with sequential and parallel orchestration patterns |
| [Critical Sections](./durable-functions/typescript/CriticalSections) | Entity Coordination | Coordinate access to entities with durable locks |

### PowerShell

| Sample | Pattern | Description |
|--------|---------|-------------|
| [Hello Cities](./durable-functions/powershell/HelloCities) | Function Chaining, Fan-out/Fan-in | PowerShell quickstart with sequential and parallel orchestration patterns |

---

## Durable Extension for Microsoft Agent Framework

The [Durable Task extension for Microsoft Agent Framework](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-microsoft-agent-framework) lets you make any [Microsoft Agent Framework](https://github.com/microsoft/agent-framework) agent durable with persistent sessions, built-in API endpoints, and distributed scaling, without changes to your agent logic. It also supports graph-based workflows via `WorkflowBuilder`.

> **Setup:** These samples live in the `microsoft/agent-framework` repo and are included here via a Git submodule. Run `git submodule update --init external/agent-framework` after cloning.

The links below open the samples at the pinned submodule revision in the upstream repository. This keeps navigation consistent with the included code even when upstream reorganizes its samples. The local sample directories are symlinks into the submodule and cannot be browsed through GitHub as ordinary directories.

📂 **[Full details and sample index →](./durable-extension-for-agent-framework/)**

### Durable Agents (.NET)

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Single Agent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/01_SingleAgent) | Azure Functions | Basic durable agent with persistent sessions |
| [Orchestration Chaining](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/02_AgentOrchestration_Chaining) | Azure Functions | Multi-agent sequential orchestration |
| [Orchestration Concurrency](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/03_AgentOrchestration_Concurrency) | Azure Functions | Parallel agent execution (fan-out/fan-in) |
| [Orchestration Conditionals](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/04_AgentOrchestration_Conditionals) | Azure Functions | Conditional routing between agents |
| [Human-in-the-Loop](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/05_AgentOrchestration_HITL) | Azure Functions | Agent pauses for human approval |
| [Reliable Streaming](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/08_ReliableStreaming) | Azure Functions | Real-time token streaming with durability |
| [Single Agent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/ConsoleApps/01_SingleAgent) | Console App | Same pattern without Azure Functions |

### Durable MAF Workflows (.NET)

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Sequential](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/AzureFunctions/01_SequentialWorkflow) | Azure Functions | Order cancellation pipeline: look up → cancel → notify |
| [Concurrent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/AzureFunctions/02_ConcurrentWorkflow) | Azure Functions | Fan-out to multiple expert agents, fan-in to aggregate |
| [Human-in-the-Loop](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/AzureFunctions/03_WorkflowHITL) | Azure Functions | Expense reimbursement with multi-stage approvals |
| [Sequential](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/ConsoleApps/01_SequentialWorkflow) | Console App | Sequential executor pipeline |
| [Conditional Edges](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/ConsoleApps/03_ConditionalEdges) | Console App | Runtime routing based on conditions |

### Durable Agents (Python)

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Single Agent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/01_single_agent) | Azure Functions | Basic durable agent with persistent sessions |
| [Multi-Agent Orchestration](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/02_multi_agent) | Azure Functions | Multiple agents in a durable orchestration |
| [Human-in-the-Loop](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/07_single_agent_orchestration_hitl) | Azure Functions | Agent pauses for human approval |
| [Reliable Streaming](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/03_reliable_streaming) | Azure Functions | Real-time token streaming with durability |
| [Single Agent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/durabletask/01_single_agent) | Durable Task SDK | Agent hosted with DT SDK directly |

---

## Scenarios

| Sample | Description |
|--------|-------------|
| [Autoscaling in ACA](./scenarios/AutoscalingInACA) | KEDA-based dynamic worker scaling in Azure Container Apps |
| [Document Processing on AKS](./scenarios/DocumentProcessingOnAKS) | .NET activity chaining and fan-out/fan-in on AKS with workload identity |
| [Work Item Filtering: .NET](./scenarios/WorkItemFilteringSplitActivities) | Route orchestration, validation, and shipping work to separate .NET workers |
| [Work Item Filtering: Java](./scenarios/WorkItemFilteringSplitActivitiesJava) | Route orchestration, validation, and shipping work to separate Java workers |
| [Work Item Filtering: Python](./scenarios/WorkItemFilteringSplitActivitiesPython) | Route orchestration, validation, and shipping work to separate Python workers |

---

## Preview Features

| Feature | Status | Samples |
|---------|--------|---------|
| [On-demand Sandboxes](../preview-features/on-demand-sandboxes/) | Private preview; access must be enabled on your scheduler | [.NET and Python samples](../preview-features/on-demand-sandboxes/samples/) |

---

## Contributing a Sample

Want to add your own sample? See the [Contributing Guide](../CONTRIBUTING.md) for guidelines on sample structure, documentation, and submission.
