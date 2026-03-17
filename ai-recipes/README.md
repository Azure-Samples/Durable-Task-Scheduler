# AI Recipes for Durable Task

**Production-ready patterns for building durable AI applications** — each recipe demonstrates a specific pattern using the [Durable Task SDK](https://github.com/microsoft/durabletask-python) with the [Azure Durable Task Scheduler](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler).

Every recipe includes an **OpenAI SDK** variant (Azure OpenAI) and/or a **GitHub Copilot SDK** variant, so you can choose the LLM integration that fits your stack.

---

## Recipes

| # | Recipe | What You'll Learn | Key Patterns |
| --- | -------- | ------------------- | -------------- |
| 01 | [Agentic Loop](./01-agentic-loop/) | Build an autonomous tool-calling agent | While-loop, dynamic activities |
| 02 | [Human-in-the-Loop](./02-human-in-the-loop/) | Pause agents for human approval | External events, durable timers |
| 03 | [Durable MCP Tools](./03-durable-mcp-tools/) | Back MCP tools with durable workflows | FastMCP + orchestrations |
| 04 | [RAG Pipeline](./04-rag-pipeline/) | Parallel retrieval + LLM generation | Fan-out/fan-in |
| 05 | [Structured Outputs](./05-structured-outputs/) | Validate LLM output with Pydantic | Retry on validation failure |
| 06 | [Deep Research](./06-deep-research/) | Multi-step iterative research agent | Sub-orchestrations, tool calling |
| 07 | [Multi-Agent](./07-multi-agent/) | Coordinate multiple AI agents | Sub-orchestrations, durable entities |
| 08 | [Durable Agent Session](./08-durable-agent-session/) | Crash-resilient multi-turn agent | Copilot SDK + session persistence |
| 09 | [Scheduled Agent](./09-scheduled-agent/) | Timer-triggered agent runs | Eternal orchestrations, continue-as-new |

---

## SDK Variants

Each recipe provides one or both SDK variants:

| Variant | LLM Provider | Best For |
| --------- | ------------- | ---------- |
| **`openai-sdk/`** | Azure OpenAI (via OpenAI Python SDK) | Direct API access, structured outputs, fine-grained control |
| **`copilot-sdk/`** | GitHub Copilot (via `github-copilot-sdk`) | GitHub-native auth, custom agents, permission handling |

> Recipes **08** and **09** are Copilot SDK–only. Recipe **03** openai-sdk is an MCP server (no standalone client). All other recipes include both variants.

---

## Prerequisites

### 1. Durable Task Scheduler Emulator

```bash
docker pull mcr.microsoft.com/dts/dts-emulator:latest
docker run -d -p 8080:8080 -p 8082:8082 --name dts-emulator mcr.microsoft.com/dts/dts-emulator:latest
```

Dashboard: [http://localhost:8082](http://localhost:8082)

### 2. Azure OpenAI (for `openai-sdk` variants)

Copy the environment template and fill in your Azure OpenAI credentials:

```bash
cp .env.example .env
# Edit .env with your values
```

```env
OPENAI_API_KEY=your-api-key-here
OPENAI_BASE_URL=https://your-resource.openai.azure.com/openai/v1/
OPENAI_MODEL=gpt-5.4
OPENAI_EMBEDDING_MODEL=text-embedding-3-small
```

### 3. GitHub Copilot SDK (for `copilot-sdk` variants)

```bash
pip install github-copilot-sdk
```

Authenticate via the [GitHub CLI](https://cli.github.com/):

```bash
gh auth login
export GITHUB_TOKEN=$(gh auth token)
```

---

## Quick Start

Pick a recipe and run it:

```bash
# Example: Recipe 01 — Agentic Loop (OpenAI SDK)
cd 01-agentic-loop/openai-sdk
pip install -r requirements.txt

# Terminal 1: Start the worker
python worker.py

# Terminal 2: Run the client
python client.py "What does 'ephemeral' mean?"
```

Open [http://localhost:8082](http://localhost:8082) to see the orchestration in the dashboard.

---

**Worker** → connects to the scheduler and processes orchestration/activity work items  
**Client** → schedules a new orchestration instance and waits for the result  
**Activities** → non-deterministic functions (LLM calls, HTTP, database)  
**Orchestrations** → deterministic workflow logic (must use `yield` for all async operations)

---

## Shared Utilities

The [`shared/`](./shared/) directory contains reusable helpers:

- **`copilot_activity.py`** — Durable Task activity wrapping the GitHub Copilot SDK (`CopilotRequest` / `CopilotResponse` dataclasses, `run_copilot_agent()`)

---

## Pattern Reference

| Pattern | Recipes | Description |
| --------- | --------- | ------------- |
| **Fan-Out / Fan-In** | 04, 06, 07 | Parallel work with aggregated results |
| **Human Interaction** | 02 | Workflow pauses for external approval via events |
| **Sub-Orchestrations** | 06, 07 | Reusable, composable workflow components |
| **Durable Entities** | 07 | Stateful objects shared across orchestrations |
| **Eternal Orchestrations** | 09 | Long-running background loops with `continue_as_new` |
| **Retry / Validation** | 01, 05 | Retry on failure with backoff; validate and re-prompt |
| **MCP Integration** | 03 | Expose durable workflows as MCP tools/tasks |
