# Durable Extension for Microsoft Agent Framework - Samples

The [Durable Task extension for Microsoft Agent Framework](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-microsoft-agent-framework) brings [durable execution](https://learn.microsoft.com/azure/durable-task/sdks/durable-task-for-ai-agents) directly into the [Microsoft Agent Framework](https://github.com/microsoft/agent-framework). You can register agents with the extension to make them automatically durable, with persistent sessions, built-in API endpoints, and distributed scaling, **without changes to your agent logic**.

The extension internally implements [entity-based agent loops](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-patterns#entity-based-agent-loops), where each agent session is a durable entity that automatically manages conversation state and checkpointing.

## What You Get

| Capability | Description |
|------------|-------------|
| **Persistent sessions** | Conversation history survives restarts, crashes, and scaling events, with no external database needed |
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

Host the agent with the Durable Task SDK directly, with no Azure Functions dependency:

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
| **Tool calling** | Agents with function tools; tool calls are checkpointed and not re-executed on recovery |
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

> **Note:** These directories are symlinks into the [`microsoft/agent-framework`](https://github.com/microsoft/agent-framework) repo, which is included as a Git submodule at `external/agent-framework`. This avoids duplicating samples across repos. GitHub cannot browse through these symlinks as ordinary directories, so the sample index links directly to the pinned upstream revision. The links and local copies use the same revision, even if upstream reorganizes its samples.

## Getting Started

### Prerequisites

1. [Docker](https://www.docker.com/products/docker-desktop/) for the Durable Task Scheduler emulator
2. [.NET 10 SDK](https://dotnet.microsoft.com/download/dotnet/10.0) (for .NET samples) or [Python 3.10+](https://www.python.org/) (for Python samples)
3. A model endpoint and credentials, such as [Azure OpenAI](https://learn.microsoft.com/azure/ai-services/openai/). Check each sample's README for its required model settings.

### First-time setup

Clone this repo with submodules to pull the sample code:

```bash
git clone --recurse-submodules https://github.com/Azure-Samples/Durable-Task-Scheduler.git
```

If you've already cloned without submodules, initialize them separately:

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

Update the local symlink targets and the sample links in this README and [the sample catalog](../README.md) in the same change. The upstream directory layout has changed since the pinned revision, so pulling `main` alone can leave the local sample paths broken.

```bash
cd external/agent-framework
git pull origin main
cd ../..
git add external/agent-framework
git commit -m "Update agent-framework submodule"
```

A [GitHub Actions workflow](../../.github/workflows/update-agent-framework.yml) also runs weekly to auto-update the submodule via PR.

## Sample Index

### .NET - Durable Agents (Hosting)

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Single Agent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/01_SingleAgent) | Azure Functions | Basic durable agent with persistent sessions |
| [Orchestration Chaining](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/02_AgentOrchestration_Chaining) | Azure Functions | Multi-agent sequential orchestration |
| [Orchestration Concurrency](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/03_AgentOrchestration_Concurrency) | Azure Functions | Parallel agent execution with fan-out/fan-in |
| [Orchestration Conditionals](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/04_AgentOrchestration_Conditionals) | Azure Functions | Conditional routing between agents |
| [Human-in-the-Loop](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/05_AgentOrchestration_HITL) | Azure Functions | Agent pauses for human approval |
| [Long-Running Tools](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/06_LongRunningTools) | Azure Functions | Durable execution of long-running agent tools |
| [MCP Server](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/07_AgentAsMcpTool) | Azure Functions | Agent as a Model Context Protocol server |
| [Reliable Streaming](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/AzureFunctions/08_ReliableStreaming) | Azure Functions | Real-time token streaming with durability |
| [Single Agent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/ConsoleApps/01_SingleAgent) | Console App | Basic durable agent without Azure Functions |
| [Orchestration Chaining](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/ConsoleApps/02_AgentOrchestration_Chaining) | Console App | Multi-agent orchestration in a console app |
| [Orchestration Concurrency](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/ConsoleApps/03_AgentOrchestration_Concurrency) | Console App | Parallel agents in a console app |
| [Orchestration Conditionals](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/ConsoleApps/04_AgentOrchestration_Conditionals) | Console App | Conditional agent routing in a console app |
| [Human-in-the-Loop](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/ConsoleApps/05_AgentOrchestration_HITL) | Console App | Human approval in a console app |
| [Long-Running Tools](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/ConsoleApps/06_LongRunningTools) | Console App | Durable execution of long-running agent tools |
| [Reliable Streaming](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableAgents/ConsoleApps/07_ReliableStreaming) | Console App | Real-time token streaming with durability |

Standalone **Custom State** (custom persistent agent state) and **Durable Agent Client** (client access to durable agents) samples are not included in the pinned submodule revision.

### .NET - Durable MAF Workflows

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Sequential](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/AzureFunctions/01_SequentialWorkflow) | Azure Functions | Order cancellation pipeline: look up → cancel → notify |
| [Concurrent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/AzureFunctions/02_ConcurrentWorkflow) | Azure Functions | Fan-out to physicist & chemist agents, fan-in to aggregate |
| [Human-in-the-Loop](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/AzureFunctions/03_WorkflowHITL) | Azure Functions | Expense reimbursement with manager + parallel finance approvals |
| [MCP Tool](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/AzureFunctions/04_WorkflowMcpTool) | Azure Functions | Workflow exposed as an MCP tool |
| [Combined](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/AzureFunctions/05_WorkflowAndAgents) | Azure Functions | Workflows + agents in the same app |
| [Sequential](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/ConsoleApps/01_SequentialWorkflow) | Console App | Sequential executor pipeline |
| [Concurrent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/ConsoleApps/02_ConcurrentWorkflow) | Console App | Fan-out/fan-in with parallel execution |
| [Conditional Edges](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/ConsoleApps/03_ConditionalEdges) | Console App | Runtime routing based on conditions |
| [Combined](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/ConsoleApps/04_WorkflowAndAgents) | Console App | Workflows + agents in the same app |
| [Events](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/ConsoleApps/05_WorkflowEvents) | Console App | Reacting to external events in workflows |
| [Shared State](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/ConsoleApps/06_WorkflowSharedState) | Console App | Passing state between workflow steps |
| [Sub-Workflows](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/ConsoleApps/07_SubWorkflows) | Console App | Composing reusable sub-workflows |
| [Human-in-the-Loop](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/dotnet/samples/04-hosting/DurableWorkflows/ConsoleApps/08_WorkflowHITL) | Console App | Workflow pauses for external approval |

A standalone **Streaming** workflow sample is not included in the pinned submodule revision. The reliable streaming agent samples above demonstrate durable token streaming.

### Python - Durable Agents (Hosting)

| Sample | Hosting | Description |
|--------|---------|-------------|
| [Single Agent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/01_single_agent) | Azure Functions | Basic durable agent with persistent sessions |
| [Multi-Agent Orchestration](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/02_multi_agent) | Azure Functions | Multiple agents coordinated in an orchestration |
| [Reliable Streaming](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/03_reliable_streaming) | Azure Functions | Real-time token streaming with durability |
| [Orchestration Chaining](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/04_single_agent_orchestration_chaining) | Azure Functions | Sequential calls to a durable agent |
| [Orchestration Concurrency](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/05_multi_agent_orchestration_concurrency) | Azure Functions | Parallel agent execution |
| [Orchestration Conditionals](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/06_multi_agent_orchestration_conditionals) | Azure Functions | Conditional routing between agents |
| [Human-in-the-Loop](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/07_single_agent_orchestration_hitl) | Azure Functions | Agent pauses for human approval |
| [MCP Server](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/08_mcp_server) | Azure Functions | Agent exposed through Model Context Protocol |
| [Workflow Shared State](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/09_workflow_shared_state) | Azure Functions | Shared state across workflow steps |
| [Workflow Without Shared State](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/10_workflow_no_shared_state) | Azure Functions | Workflow steps without shared state |
| [Parallel Workflow](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/11_workflow_parallel) | Azure Functions | Concurrent workflow steps |
| [Workflow Human-in-the-Loop](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/azure_functions/12_workflow_hitl) | Azure Functions | Workflow pauses for external approval |
| [Single Agent](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/durabletask/01_single_agent) | Durable Task SDK | Agent hosted with the DT SDK directly |
| [Multi-Agent Orchestration](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/durabletask/02_multi_agent) | Durable Task SDK | Multi-agent orchestration without Azure Functions |
| [Reliable Streaming](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/durabletask/03_single_agent_streaming) | Durable Task SDK | Durable token streaming without Azure Functions |
| [Orchestration Chaining](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/durabletask/04_single_agent_orchestration_chaining) | Durable Task SDK | Sequential calls to a durable agent |
| [Orchestration Concurrency](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/durabletask/05_multi_agent_orchestration_concurrency) | Durable Task SDK | Parallel agent execution |
| [Orchestration Conditionals](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/durabletask/06_multi_agent_orchestration_conditionals) | Durable Task SDK | Conditional routing between agents |
| [Human-in-the-Loop](https://github.com/microsoft/agent-framework/tree/3e54a689fc96d681a072fe7e7cfc445909dac74b/python/samples/04-hosting/durabletask/07_single_agent_orchestration_hitl) | Durable Task SDK | Human approval with the DT SDK |

Agents can use function tools, but this revision does not include a standalone Python **Tool Calling** sample for either hosting option.

## Learn More

- [Durable Task extension for Microsoft Agent Framework](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-microsoft-agent-framework) - Full documentation
- [Durable Task for AI Agents](https://learn.microsoft.com/azure/durable-task/sdks/durable-task-for-ai-agents) - Why durable execution matters for AI agents
- [Agentic Application Patterns](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-patterns) - All supported patterns
- [Microsoft Agent Framework](https://github.com/microsoft/agent-framework) - The agent framework itself
- [Durable Task Scheduler Dashboard](https://learn.microsoft.com/azure/durable-task/scheduler/durable-task-scheduler-dashboard) - Monitor agents, orchestrations, and workflows
