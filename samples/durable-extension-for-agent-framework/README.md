# Durable Extension for Microsoft Agent Framework — Samples

The [Durable Task extension for Microsoft Agent Framework](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-microsoft-agent-framework) brings [durable execution](https://learn.microsoft.com/azure/durable-task/sdks/durable-task-for-ai-agents) directly into the [Microsoft Agent Framework](https://github.com/microsoft/agent-framework). You can register agents with the extension to make them automatically durable — with persistent sessions, built-in API endpoints, and distributed scaling — **without changes to your agent logic**.

The extension internally implements [entity-based agent loops](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-patterns#entity-based-agent-loops), where each agent session is a durable entity that automatically manages conversation state and checkpointing.

## What You Get

| Capability | Description |
|------------|-------------|
| **Persistent sessions** | Conversation history survives restarts, crashes, and scaling events — no external database needed |
| **Automatic checkpointing** | Every agent interaction and tool call is checkpointed; completed work is never re-executed on recovery |
| **Built-in HTTP endpoints** | Send messages, check status, and manage sessions via auto-generated REST APIs |
| **Multi-agent orchestration** | Coordinate multiple specialized agents as steps in a durable orchestration with automatic recovery |
| **Graph-based workflows** | Define multi-step pipelines of executors and agents using `WorkflowBuilder` with automatic checkpointing |
| **Session TTL** | Automatic cleanup of idle sessions to manage storage and costs |
| **Two hosting options** | Azure Functions (serverless) or bring-your-own compute (console apps, containers, etc.) |

## Hosting Approaches

### Azure Functions

One line to make your agent durable with serverless hosting:

```csharp
using IHost app = FunctionsApplication
    .CreateBuilder(args)
    .ConfigureFunctionsWebApplication()
    .ConfigureDurableAgents(options => options.AddAIAgent(agent))
    .Build();
app.Run();
```

### Bring Your Own Compute (Console Apps, Containers, etc.)

Host the agent with the Durable Task SDK directly — no Azure Functions dependency:

```csharp
IHost host = Host.CreateDefaultBuilder(args)
    .ConfigureServices(services =>
    {
        services.ConfigureDurableAgents(
            options => options.AddAIAgent(agent),
            workerBuilder: builder => builder.UseDurableTaskScheduler(connectionString),
            clientBuilder: builder => builder.UseDurableTaskScheduler(connectionString));
    })
    .Build();
await host.StartAsync();
```

## Patterns Demonstrated

### Durable Agents (Hosting)

These samples show how to host agents with the Durable Task extension. Each agent session is a durable entity that persists conversation history, supports tool calling, and recovers automatically from failures.

| Pattern | Description |
|---------|-------------|
| **Single agent** | One LLM agent with persistent sessions and built-in HTTP endpoints |
| **Multi-agent orchestration** | Multiple specialized agents coordinated as checkpointed steps in a durable orchestration |
| **Tool calling** | Agents with function tools — tool calls are checkpointed and not re-executed on recovery |
| **MCP server** | Agent hosted as a [Model Context Protocol](https://modelcontextprotocol.io/) server |
| **Reliable streaming** | Real-time token streaming with durable delivery guarantees |
| **Human-in-the-loop** | Agents that pause for human approval before continuing |

### Durable MAF Workflows

These samples show how to use the [Microsoft Agent Framework `WorkflowBuilder`](https://learn.microsoft.com/agent-framework/workflows) with the Durable Task extension. The extension automatically checkpoints each step in the graph and recovers from failures without changes to the workflow definition.

| Pattern | Description |
|---------|-------------|
| **Sequential** | Chain executors into a multi-step pipeline (e.g., look up → cancel → notify) |
| **Fan-out/fan-in** | Run multiple agents or executors in parallel, then aggregate results |
| **Conditional routing** | Route execution to different branches based on runtime results (e.g., spam detection) |
| **Human-in-the-loop** | Pause workflow execution at designated points to wait for external approval |
| **Sub-workflows** | Compose complex workflows from reusable sub-workflows |
| **Shared state** | Pass state between workflow steps using context |
| **Events** | React to external events during workflow execution |

## Sample Structure

```
python/
  hosting/
    azure-functions/   → Azure Functions agent hosting samples
    durable-task/      → Durable Task SDK agent hosting samples (no Azure Functions)
dotnet/
  hosting/
    azure-functions/   → Azure Functions agent hosting samples
    console-apps/      → Console app agent hosting samples
  durable-maf-workflows/
    azure-functions/   → Graph-based workflow samples (Azure Functions)
    console-apps/      → Graph-based workflow samples (Console Apps)
```

> **Note:** These directories are symlinks into the [`microsoft/agent-framework`](https://github.com/microsoft/agent-framework) repo, which is included as a Git submodule at `external/agent-framework`. This avoids duplicating samples across repos.

## Getting Started

### Prerequisites

1. [Docker](https://www.docker.com/products/docker-desktop/) — for the Durable Task Scheduler emulator
2. [.NET 10 SDK](https://dotnet.microsoft.com/download/dotnet/10.0) (for .NET samples) or [Python 3.10+](https://www.python.org/) (for Python samples)
3. (Optional) [Azure OpenAI](https://learn.microsoft.com/azure/ai-services/openai/) endpoint for real LLM integration

### First-time setup

After cloning this repo, initialize the submodule to pull the sample code:

```bash
git submodule update --init external/agent-framework
```

Then start the Durable Task Scheduler emulator:

```bash
docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
```

The dashboard is available at [http://localhost:8082](http://localhost:8082).

### Updating to latest

To pull the latest samples from the agent-framework repo:

```bash
cd external/agent-framework
git pull origin main
cd ../..
git add external/agent-framework
git commit -m "Update agent-framework submodule"
```

A [GitHub Actions workflow](../../.github/workflows/update-agent-framework.yml) also runs weekly to auto-update the submodule via PR.

## Sample Index

### .NET — Durable Agents (Hosting)

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Single Agent](dotnet/hosting/azure-functions/01-SingleAgent) | Azure Functions | Basic durable agent with persistent sessions |
| [Orchestration Chaining](dotnet/hosting/azure-functions/02-OrchestrationChaining) | Azure Functions | Multi-agent sequential orchestration |
| [Orchestration Concurrency](dotnet/hosting/azure-functions/03-OrchestrationConcurrency) | Azure Functions | Parallel agent execution with fan-out/fan-in |
| [Orchestration Conditionals](dotnet/hosting/azure-functions/04-OrchestrationConditionals) | Azure Functions | Conditional routing between agents |
| [Human-in-the-Loop](dotnet/hosting/azure-functions/05-HumanInTheLoop) | Azure Functions | Agent pauses for human approval |
| [ReliableStreaming](dotnet/hosting/azure-functions/06-ReliableStreaming) | Azure Functions | Real-time token streaming with durability |
| [MCP Server](dotnet/hosting/azure-functions/07-McpServer) | Azure Functions | Agent as a Model Context Protocol server |
| [Custom State](dotnet/hosting/azure-functions/08-CustomState) | Azure Functions | Agent with custom persistent state |
| [Single Agent](dotnet/hosting/console-apps/01-SingleAgent) | Console App | Basic durable agent without Azure Functions |
| [Orchestration Chaining](dotnet/hosting/console-apps/02-OrchestrationChaining) | Console App | Multi-agent orchestration in a console app |
| [Orchestration Concurrency](dotnet/hosting/console-apps/03-OrchestrationConcurrency) | Console App | Parallel agents in a console app |
| [Orchestration Conditionals](dotnet/hosting/console-apps/04-OrchestrationConditionals) | Console App | Conditional agent routing in a console app |
| [Human-in-the-Loop](dotnet/hosting/console-apps/05-HumanInTheLoop) | Console App | Human approval in a console app |
| [Custom State](dotnet/hosting/console-apps/06-CustomState) | Console App | Custom agent state in a console app |
| [Durable Agent Client](dotnet/hosting/console-apps/07-DurableAgentClient) | Console App | Client for interacting with durable agents |

### .NET — Durable MAF Workflows

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Sequential](dotnet/durable-maf-workflows/azure-functions/01-Sequential) | Azure Functions | Order cancellation pipeline: look up → cancel → notify |
| [Concurrent](dotnet/durable-maf-workflows/azure-functions/02-Concurrent) | Azure Functions | Fan-out to physicist & chemist agents, fan-in to aggregate |
| [Human-in-the-Loop](dotnet/durable-maf-workflows/azure-functions/03-HumanInTheLoop) | Azure Functions | Expense reimbursement with manager + parallel finance approvals |
| [MCP Tool](dotnet/durable-maf-workflows/azure-functions/04-McpTool) | Azure Functions | Workflow exposed as an MCP tool |
| [Combined](dotnet/durable-maf-workflows/azure-functions/05-Combined) | Azure Functions | Workflows + agents in the same app |
| [Sequential](dotnet/durable-maf-workflows/console-apps/01-Sequential) | Console App | Sequential executor pipeline |
| [Concurrent](dotnet/durable-maf-workflows/console-apps/02-Concurrent) | Console App | Fan-out/fan-in with parallel execution |
| [Conditional Edges](dotnet/durable-maf-workflows/console-apps/03-ConditionalEdges) | Console App | Runtime routing based on conditions |
| [Events](dotnet/durable-maf-workflows/console-apps/04-Events) | Console App | Reacting to external events in workflows |
| [Shared State](dotnet/durable-maf-workflows/console-apps/05-SharedState) | Console App | Passing state between workflow steps |
| [Sub-Workflows](dotnet/durable-maf-workflows/console-apps/06-SubWorkflows) | Console App | Composing reusable sub-workflows |
| [Human-in-the-Loop](dotnet/durable-maf-workflows/console-apps/07-HumanInTheLoop) | Console App | Workflow pauses for external approval |
| [Streaming](dotnet/durable-maf-workflows/console-apps/08-Streaming) | Console App | Streaming workflow events |

### Python — Durable Agents (Hosting)

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Single Agent](python/hosting/azure-functions/01-single-agent) | Azure Functions | Basic durable agent with persistent sessions |
| [Multi-Agent Orchestration](python/hosting/azure-functions/02-multi-agent-orchestration) | Azure Functions | Multiple agents coordinated in an orchestration |
| [Tool Calling](python/hosting/azure-functions/03-tool-calling) | Azure Functions | Agent with function tools |
| [Human-in-the-Loop](python/hosting/azure-functions/04-human-in-the-loop) | Azure Functions | Agent pauses for human approval |
| [Single Agent](python/hosting/durable-task/01-single-agent) | Durable Task SDK | Agent hosted with the DT SDK directly |
| [Multi-Agent Orchestration](python/hosting/durable-task/02-multi-agent-orchestration) | Durable Task SDK | Multi-agent orchestration without Azure Functions |
| [Tool Calling](python/hosting/durable-task/03-tool-calling) | Durable Task SDK | Tool-calling agent with the DT SDK |
| [Human-in-the-Loop](python/hosting/durable-task/04-human-in-the-loop) | Durable Task SDK | Human approval with the DT SDK |

## Learn More

- [Durable Task extension for Microsoft Agent Framework](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-microsoft-agent-framework) — Full documentation
- [Durable Task for AI Agents](https://learn.microsoft.com/azure/durable-task/sdks/durable-task-for-ai-agents) — Why durable execution matters for AI agents
- [Agentic Application Patterns](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-patterns) — All supported patterns
- [Microsoft Agent Framework](https://github.com/microsoft/agent-framework) — The agent framework itself
- [Durable Task Scheduler Dashboard](https://learn.microsoft.com/azure/durable-task/scheduler/durable-task-scheduler-dashboard) — Monitor agents, orchestrations, and workflows
