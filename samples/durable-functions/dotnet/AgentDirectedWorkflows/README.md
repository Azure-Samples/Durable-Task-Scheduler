# Agent-Directed Workflows — Durable Functions (.NET)

This sample demonstrates how to build an **agent-directed workflow** (agent loop) using [durable entities](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-entities) in [Azure Durable Functions](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview) with the [Durable Task Scheduler](https://learn.microsoft.com/azure/durable-task/overview).

It corresponds to the **"Build your own using orchestrations and entities"** approach described in the [Durable Task for AI Agents](https://learn.microsoft.com/azure/durable-task/sdks/durable-task-for-ai-agents) comparison table, using the **entity-based agent loop** pattern from [Agentic Application Patterns](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-patterns#entity-based-agent-loops).

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
│  HTTP POST /api/chat/{sessionId}                      │
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
├── ChatAgentEntity.cs      # The durable entity (agent) and HTTP endpoints
├── Program.cs              # Host setup: IChatClient + Redis registration
├── Models/
│   └── Models.cs           # Data types: ChatMsg, ChatRequest, ChatAgentState, ChatTool
├── Tools/
│   └── AgentTools.cs       # Tool definitions and execution logic
├── host.json               # Durable Functions config (storage provider, connection string)
└── local.settings.json     # Connection strings, Redis, and Azure OpenAI settings
```

### Key Files Explained

**`ChatAgentEntity.cs`** — Contains two things:
- `ChatAgentEntity` — The durable entity. Its `Message()` method is the agent loop: it streams from the LLM via `GetStreamingResponseAsync`, publishes text chunks to Redis, checks for tool calls, and repeats until the LLM gives a final text reply. State (conversation history) is persisted automatically.
- `ChatEndpoints` — HTTP triggers that expose the agent as a streaming API. The `SendMessage` endpoint sets up an SSE response, subscribes to the Redis channel, signals the entity, and streams chunks back.

**`Tools/AgentTools.cs`** — Defines what tools the LLM can call. Currently includes `get_weather` (with a hardcoded response). To add new capabilities, add a definition and an execution case. The LLM discovers available tools automatically.

**`Program.cs`** — Registers the `IChatClient` implementation via [Microsoft.Extensions.AI](https://learn.microsoft.com/dotnet/ai/microsoft-extensions-ai) and the Redis `IConnectionMultiplexer`. If Azure OpenAI environment variables are set, it uses a real LLM. Otherwise, it falls back to a simple echo client for local development.

## Prerequisites

1. [.NET 10 SDK](https://dotnet.microsoft.com/download/dotnet/10.0) or later
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the Durable Task Scheduler emulator and Redis)
3. [Azure Functions Core Tools](https://learn.microsoft.com/azure/azure-functions/functions-run-local) v4

## Running the Sample

### 1. Start the Durable Task Scheduler emulator and Redis

```bash
# Durable Task Scheduler emulator
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest

# Redis
docker run --name redis -d -p 6379:6379 redis:latest
```

### 2. (Optional) Configure Azure OpenAI

Update `local.settings.json` to use a real LLM instead of the echo fallback:

```json
{
  "Values": {
    "AZURE_OPENAI_ENDPOINT": "https://<your-resource>.openai.azure.com",
    "AZURE_OPENAI_DEPLOYMENT": "<your-deployment-name>"
  }
}
```

The sample uses `DefaultAzureCredential` for authentication — make sure you're signed in via `az login`.

### 3. Run the function app

```bash
func start
```

### 4. Chat with the agent

```bash
# Streaming (default) — response streams as Server-Sent Events
curl -N -X POST http://localhost:7071/api/chat/session1 \
  -H "Content-Type: application/json" \
  -d '{"message":"What is the weather in Seattle?"}'

# SSE output:
# data: {"type":"chunk","content":"It's"}
# data: {"type":"chunk","content":" 72°F"}
# data: {"type":"chunk","content":" and sunny"}
# data: {"type":"chunk","content":" in Seattle."}
# data: {"type":"done"}

# Non-streaming — waits for the full response and returns JSON
curl -X POST "http://localhost:7071/api/chat/session1?stream=false" \
  -H "Content-Type: application/json" \
  -d '{"message":"What is the weather in Seattle?"}'

# JSON output:
# {"sessionId":"session1","message":"It's 72°F and sunny in Seattle."}

# View the full conversation history
curl http://localhost:7071/api/chat/session1/history

# Reset the conversation
curl -X POST http://localhost:7071/api/chat/session1/reset
```

> **Tip:** Use `curl -N` (no buffering) to see SSE events in real-time when streaming.

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/chat/{sessionId}` | Send a message and stream the reply as SSE (default). Add `?stream=false` for a JSON response. Body: `{"message": "..."}` |
| GET | `/api/chat/{sessionId}/history` | Get the full conversation history for a session |
| POST | `/api/chat/{sessionId}/reset` | Clear a session's conversation history |

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
- [Durable Entities](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-entities) — Deep dive on the entity programming model
- [Microsoft.Extensions.AI](https://learn.microsoft.com/dotnet/ai/microsoft-extensions-ai) — The AI abstraction layer used for LLM integration
