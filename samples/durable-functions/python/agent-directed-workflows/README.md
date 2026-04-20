# Agent-Directed Workflows — Durable Functions (Python)

This sample demonstrates how to build an **agent-directed workflow** (agent loop) using [durable entities](https://learn.microsoft.com/azure/durable-functions/durable-functions-entities) in [Azure Durable Functions for Python](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview?tabs=isolated-process%2Cnodejs-v3&pivots=python) backed by the [Durable Task Scheduler](https://learn.microsoft.com/azure/durable-task/overview).

It corresponds to the **"Build your own using orchestrations and entities"** approach described in the [Durable Task for AI Agents](https://learn.microsoft.com/azure/durable-task/sdks/durable-task-for-ai-agents) comparison table.

> **Note:** A [Durable Task SDK version](../../../durable-task-sdks/python/agent-directed-workflows/) of this sample is also available. That version uses FastAPI with SSE streaming and gives you full control over the web host. This version uses the standard Azure Functions HTTP model (non-streaming responses).

## What Is an Agent-Directed Workflow?

In a typical deterministic workflow, *your code* controls the execution path. In an **agent-directed workflow**, the **LLM drives the control flow** — it decides which tools to call, in what order, and when the task is complete. The execution path isn't known until runtime.

## How This Sample Works

Each chat session is a **durable entity** that holds the full conversation history. When you send a message, the entity runs the agent loop (call LLM → execute tools → repeat) and streams response chunks to Redis pub/sub. The HTTP function subscribes to the Redis channel and collects the full response before returning it as JSON.

### Project Structure

```
agent-directed-workflows/
├── function_app.py           # Entity, HTTP functions, and agent loop logic
├── tools.py                  # Tool definitions and execution logic
├── host.json                 # Functions host configuration (Durable Task Scheduler)
├── requirements.txt          # Python dependencies
└── README.md
```

## Prerequisites

1. [Python 3.10+](https://www.python.org/downloads/)
2. [Azure Functions Core Tools v4+](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
3. [Docker](https://www.docker.com/products/docker-desktop/) (for the Durable Task Scheduler emulator and Redis)

## Running the Sample

### 1. Start the Durable Task Scheduler emulator and Redis

```bash
# Durable Task Scheduler emulator
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest

# Redis
docker run --name redis -d -p 6379:6379 redis:latest
```

### 2. Install dependencies

```bash
pip install -r requirements.txt
```

### 3. Configure local settings

Create a `local.settings.json` file:

```json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "FUNCTIONS_WORKER_RUNTIME": "python",
    "DTS_CONNECTION_STRING": "Endpoint=http://localhost:8080;Authentication=None",
    "TASKHUB_NAME": "default",
    "REDIS_CONNECTION_STRING": "localhost:6379"
  }
}
```

### 4. (Optional) Configure Azure OpenAI

Add these to your `local.settings.json` `Values` to use a real LLM instead of the echo fallback:

```json
{
  "AZURE_OPENAI_ENDPOINT": "https://<your-resource>.openai.azure.com",
  "AZURE_OPENAI_DEPLOYMENT": "<your-deployment-name>"
}
```

The sample uses `DefaultAzureCredential` — make sure you're signed in via `az login`.

### 5. Run the function app

```bash
func start
```

### 6. Chat with the agent

```bash
# Send a message (returns JSON with full response)
curl -X POST http://localhost:7071/api/chat/session1 \
  -H "Content-Type: application/json" \
  -d '{"message":"What is the weather in Seattle?"}'

# View conversation history
curl http://localhost:7071/api/chat/session1/history

# Reset the conversation
curl -X POST http://localhost:7071/api/chat/session1/reset
```

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/chat/{sessionId}` | Send a message and get the full response as JSON. Body: `{"message": "..."}` |
| GET | `/api/chat/{sessionId}/history` | Get the full conversation history for a session |
| POST | `/api/chat/{sessionId}/reset` | Clear a session's conversation history |

## Key Differences from the Durable Task SDK Version

| Aspect | Durable Task SDK | Durable Functions |
|--------|-----------------|-------------------|
| **Web host** | FastAPI (full control) | Azure Functions HTTP triggers |
| **Streaming** | SSE via `StreamingResponse` | Non-streaming (JSON response) |
| **Entity registration** | `worker.add_entity()` | `@bp.entity_trigger` decorator |
| **Client** | `DurableTaskSchedulerClient` | `DurableOrchestrationClient` |
| **Configuration** | Environment variables | `host.json` + `local.settings.json` |

## Learn More

- [Durable Task for AI Agents](https://learn.microsoft.com/azure/durable-task/sdks/durable-task-for-ai-agents)
- [Agentic Application Patterns](https://learn.microsoft.com/azure/durable-task/sdks/durable-agents-patterns)
- [Durable Entities in Azure Functions](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-entities)
- [Durable Task Scheduler](https://learn.microsoft.com/azure/durable-task/overview)
