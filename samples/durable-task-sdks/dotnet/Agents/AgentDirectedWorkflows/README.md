# Agent-Directed Workflows — Durable Task SDK (.NET)

This sample demonstrates how to build an **agent-directed workflow** (agent loop) using [durable entities](https://learn.microsoft.com/azure/durable-task/concepts/entities) from the [Durable Task SDK for .NET](https://github.com/microsoft/durabletask-dotnet) with the [Durable Task Scheduler](https://learn.microsoft.com/azure/durable-task/overview).

It corresponds to the **"Build your own using orchestrations and entities"** approach described in the [Durable Task for AI Agents](https://learn.microsoft.com/azure/durable-task/sdks/durable-task-for-ai-agents) comparison table, using the **entity-based agent loop** pattern from [Agentic Application Patterns](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-patterns#entity-based-agent-loops).

> **Note:** A [Durable Functions version](../../../durable-functions/dotnet/AgentDirectedWorkflows/) of this sample is also available. This version uses the Durable Task SDK directly (no Azure Functions dependency), which gives you full control over the web host and API surface.

## What Is an Agent-Directed Workflow?

In a typical deterministic workflow, *your code* controls the execution path — you define the sequence of steps ahead of time. In an **agent-directed workflow** (also called an agent loop), the **LLM drives the control flow**. You provide tools and instructions, but the agent decides which tools to call, in what order, and when the task is complete. The execution path isn't known until runtime.

This pattern is well suited for conversational agents with tool-calling capabilities, open-ended research tasks, and any scenario where the number and order of steps can't be predicted.

## How This Sample Works

### The Core Idea

Each chat session is a **durable entity** — a long-lived, stateful actor managed by the Durable Task Scheduler. The entity holds the full conversation history in its state. When you send it a message, it runs the agent loop internally and **streams the LLM's response in real-time** via Redis pub/sub:

```
User: "What's the weather in Seattle?"
  │
  ▼
┌──────────────────────────────────────────────────────┐
│  HTTP POST /chat/{sessionId}                          │
│                                                       │
│  1. Subscribe to Redis channel                        │
│  2. Signal entity (fire-and-forget)                   │
│  3. Stream SSE events from Redis to client            │
└────────────────────┬─────────────────────────────────┘
                     │ (signal)
                     ▼
┌──────────────────────────────────────────────────────┐
│  ChatAgentEntity (durable entity)                     │
│                                                       │
│  State = conversation history  ← auto-persisted       │
│                                                       │
│  1. Add user message to state                         │
│  2. Stream LLM response                               │
│  3. LLM returns tool calls?                           │
│     ├─ YES → execute tools, go to 2                   │
│     └─ NO  → publish chunks to Redis as they arrive   │
│  4. Save full reply to state                          │
│  5. Publish "done" event to Redis                     │
└──────────────────────────────────────────────────────┘
```

**Key properties:**
- **Durable state**: The entity's conversation history survives restarts, crashes, and scaling events.
- **Real-time streaming**: LLM response tokens are published to Redis and forwarded to the client as SSE events as they're generated.
- **No orchestration bridge**: The HTTP layer signals the entity (fire-and-forget) and reads the response from Redis — no intermediate orchestration needed.

### Why Entities?

Durable entities are a natural fit for AI agents because:

- **Long-lived state**: An entity persists for as long as you need it. A chat session can last minutes, days, or weeks.
- **Automatic persistence**: You just read and write `State` — the framework handles serialization and storage.
- **Actor model**: Each entity processes one operation at a time, so there are no concurrency issues with the conversation history.
- **Addressable**: Each entity has a unique ID (the session ID), making it easy to route messages to the right agent.

### Streaming via Redis

Instead of request-response (which blocks until the full LLM reply is generated), this sample uses **Redis pub/sub** to stream response chunks in real-time:

1. The HTTP handler subscribes to a Redis channel **before** signaling the entity
2. The entity runs the agent loop and publishes each token to Redis as the LLM generates it
3. The HTTP handler forwards each chunk to the client as a Server-Sent Event (SSE)
4. When the entity finishes, it publishes a `done` event

Each request uses a unique `correlationId` in the channel name to isolate concurrent conversations.

### Project Structure

```
AgentDirectedWorkflows/
├── ChatAgentEntity.cs      # The durable entity (agent)
├── Program.cs              # Web host, SSE endpoints, DI registration, DTS connection
├── Models/
│   └── Models.cs           # Data types: ChatMsg, ChatRequest, ChatAgentState, ChatTool
├── Tools/
│   └── AgentTools.cs       # Tool definitions and execution logic
└── appsettings.json        # Default configuration
```

### Key Files Explained

**`ChatAgentEntity.cs`** — The durable entity. Its `Message()` method is the agent loop: it streams from the LLM via `GetStreamingResponseAsync`, publishes text chunks to Redis, checks for tool calls, and repeats until the LLM gives a final text reply. State (conversation history) is persisted automatically.

**`Program.cs`** — A standard ASP.NET minimal API that:
1. Registers the `IChatClient` (Azure OpenAI if configured, echo fallback otherwise)
2. Registers the Redis `IConnectionMultiplexer` for pub/sub streaming
3. Registers the Durable Task worker with the entity
4. Defines SSE streaming endpoints for interacting with agent sessions

**`Tools/AgentTools.cs`** — Defines what tools the LLM can call. Currently includes `get_weather` (with a hardcoded response). To add new capabilities, add a definition and an execution case. The LLM discovers available tools automatically.

## Prerequisites

1. [.NET 10 SDK](https://dotnet.microsoft.com/download/dotnet/10.0) or later
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the Durable Task Scheduler emulator and Redis)

## Running the Sample

### 1. Start the Durable Task Scheduler emulator and Redis

```bash
# Durable Task Scheduler emulator
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest

# Redis
docker run --name redis -d -p 6379:6379 redis:latest
```

### 2. (Optional) Configure Azure OpenAI

Set environment variables to use a real LLM instead of the echo fallback:

```bash
export AZURE_OPENAI_ENDPOINT="https://<your-resource>.openai.azure.com"
export AZURE_OPENAI_DEPLOYMENT="<your-deployment-name>"
```

The sample uses `DefaultAzureCredential` for authentication — make sure you're signed in via `az login`.

### 3. Run the app

```bash
dotnet run
```

The app connects to the local emulator at `http://localhost:8080` by default.

### 4. Chat with the agent

```bash
# Streaming (default) — response streams as Server-Sent Events
curl -N -X POST http://localhost:5000/chat/session1 \
  -H "Content-Type: application/json" \
  -d '{"message":"What is the weather in Seattle?"}'

# SSE output:
# data: {"type":"chunk","content":"It's"}
# data: {"type":"chunk","content":" 72°F"}
# data: {"type":"chunk","content":" and sunny"}
# data: {"type":"chunk","content":" in Seattle."}
# data: {"type":"done"}

# Non-streaming — waits for the full response and returns JSON
curl -X POST "http://localhost:5000/chat/session1?stream=false" \
  -H "Content-Type: application/json" \
  -d '{"message":"What is the weather in Seattle?"}'

# JSON output:
# {"sessionId":"session1","message":"It's 72°F and sunny in Seattle."}

# View the full conversation history
curl http://localhost:5000/chat/session1/history

# Reset the conversation
curl -X POST http://localhost:5000/chat/session1/reset
```

> **Tip:** Use `curl -N` (no buffering) to see SSE events in real-time when streaming.

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/chat/{sessionId}` | Send a message and stream the reply as SSE (default). Add `?stream=false` for a JSON response. Body: `{"message": "..."}` |
| GET | `/chat/{sessionId}/history` | Get the full conversation history for a session |
| POST | `/chat/{sessionId}/reset` | Clear a session's conversation history |

### SSE Event Format

Each SSE event is a JSON object with a `type` field:

| Type | Description | Example |
|------|-------------|---------|
| `chunk` | A token from the LLM response | `{"type":"chunk","content":"Hello"}` |
| `done` | The response is complete | `{"type":"done"}` |
| `error` | An error occurred | `{"type":"error","content":"..."}` |

## Adding New Tools

To give the agent new capabilities, edit `Tools/AgentTools.cs`:

1. Add a tool definition to the `Definitions` array
2. Add a case to the `Execute()` switch

```csharp
// In Definitions:
new("search_web", "Search the web for information"),

// In Execute():
"search_web" => SearchTheWeb(args),
```

The LLM will automatically discover the new tool and call it when appropriate.

## Learn More

- [Durable Task for AI Agents](https://learn.microsoft.com/azure/durable-task/sdks/durable-task-for-ai-agents) — Overview of how durable execution solves production challenges for AI agents
- [Agentic Application Patterns](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-patterns) — All supported patterns (deterministic workflows, agent loops, orchestration-based vs entity-based)
- [Durable Entities](https://learn.microsoft.com/azure/durable-task/concepts/entities) — Deep dive on the entity programming model
- [Microsoft.Extensions.AI](https://learn.microsoft.com/dotnet/ai/microsoft-extensions-ai) — The AI abstraction layer used for LLM integration
