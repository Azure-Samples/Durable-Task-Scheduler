# Durable MCP Tools — GitHub Repository Inspector

Build an MCP server where every tool call is backed by a durable Durable Task workflow. This recipe uses GitHub repository inspection as a practical, developer-focused example.

This recipe includes two durable implementations:

- `openai-sdk/` for the worker, orchestrations, and configuration used by the durable MCP server.
- `copilot-sdk/` for a Copilot-powered client that talks to the same recipe-level MCP server.

Note: this recipe keeps the reusable MCP server at the recipe root as `mcp_server.py`, so both variants can point at the same durable GitHub inspector tools.

## What this recipe demonstrates

This recipe demonstrates a GitHub-focused inspector that feels natural in developer workflows:

- `inspect_repo(owner, repo)` → fetch durable repository metadata such as stars, forks, language, license, and description.
- `recent_activity(owner, repo)` → runs a multi-step workflow that gathers recent commits, issues, and pull requests before returning one combined activity summary.

## Architecture

```text
MCP client
    │
    │ MCP tool invocation
    ▼
FastMCP server (mcp_server.py)
    │
    │ schedule_new_orchestration + wait_for_orchestration_completion
    ▼
Durable Task orchestrations (github_workflows.py)
    │
    │ call_activity(..., retry_policy=GITHUB_RETRY_POLICY)
    ▼
GitHub API activity (fetch_github_api)
    │
    │ GET https://api.github.com/...
    ▼
GitHub REST API
```

## Why Durable Task fits MCP tools

The MCP server stays intentionally thin:

- `mcp_server.py` accepts tool calls from any MCP-compatible client.
- Each tool starts a durable orchestration and waits for completion.
- Orchestrations own the workflow logic.
- Activities own all external I/O, including GitHub API calls and rate-limit handling.

That keeps the MCP surface small while Durable Task provides retries, persistence, and recovery.

## openai-sdk variant

Main directory: `openai-sdk/` plus recipe-root `mcp_server.py`

### Files

- `../mcp_server.py` — recipe-level FastMCP server exposing the GitHub inspector tools.
- `worker.py` — Durable Task worker that hosts orchestrations and activities.
- `orchestrations/github_workflows.py` — `GetRepoInfo` and `GetRecentActivity` orchestrations.
- `activities/github_api.py` — authenticated GitHub REST API activity with headers and rate-limit-aware errors.
- `requirements.txt` — Python dependencies.

## Copilot SDK variant

Directory: `copilot-sdk/`

### Files

- `mcp_consumer.py` — creates a Copilot SDK session that is configured with the durable GitHub MCP server.
- `client.py` — simple CLI entry point for asking repository questions through Copilot.
- `worker.py` — runs the durable GitHub inspector worker for the MCP-backed tools.
- `orchestrations.py` — durable repo inspection workflows used by the MCP server.
- `activities.py` — GitHub REST API activity implementation shared by the durable worker.

## Running the recipe

### 1. Start the Durable Task Scheduler emulator

```bash
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 \
  mcr.microsoft.com/dts/dts-emulator:latest
```

The dashboard is available at http://localhost:8082.

### 2. Install dependencies

```bash
cd ai-recipes/03-durable-mcp-tools/openai-sdk
python3 -m pip install -r requirements.txt
```

### 3. Export an optional GitHub token

Authenticated requests get a much higher GitHub API rate limit.

```bash
export GITHUB_TOKEN="your-github-token"
```

The sample still works without a token, but the activity returns a helpful error if the anonymous rate limit is exhausted.

### 4. Start the durable worker

In Terminal 1:

```bash
cd ai-recipes/03-durable-mcp-tools/openai-sdk
python3 worker.py
```

This registers `GetRepoInfo`, `GetRecentActivity`, and the shared `fetch_github_api` activity with the DTS emulator.

### 5. Connect an MCP client

Point any MCP-compatible client at the server. For example, using the `client.py` in the Copilot SDK variant:

```bash
cd ai-recipes/03-durable-mcp-tools/copilot-sdk
python3 client.py "Tell me about the microsoft/durabletask-python repo."
```

Or configure any MCP host to launch the server over stdio:

```json
{
  "mcpServers": {
    "github-inspector": {
      "command": "python3",
      "args": ["path/to/ai-recipes/03-durable-mcp-tools/mcp_server.py"],
      "env": {
        "DTS_ENDPOINT": "http://localhost:8080",
        "DTS_TASKHUB": "default",
        "GITHUB_TOKEN": "your-github-token"
      }
    }
  }
}
```

### 6. Try the tools

Ask your MCP client to use the tools, for example:

- “Tell me about the `microsoft/durabletask-python` repo.”
- “What's been happening in `facebook/react` lately?”
- “Inspect `psf/requests` and summarize the repo.”
- “Give me recent activity for `tiangolo/fastapi`.”

## Workflow details

### `GetRepoInfo`

- Fetches `GET /repos/{owner}/{repo}`.
- Formats repository metadata into a concise MCP-friendly response.
- Highlights description, language, license, stars, forks, watchers, and repo URL.

### `GetRecentActivity`

- Fans out to GitHub search endpoints for recent commits, issues, and pull requests.
- Waits for all three durable activity calls to complete.
- Fans in the results into a single activity summary for the caller.

This makes the second tool a clearer example of a multi-step durable pattern than a thin single-request wrapper.

## MCP Tasks Support

**MCP Tasks** are the protocol-level way for an MCP client to start long-running work, get back a task handle immediately, and then poll for status or fetch the final result later. They matter for durable tools because they let the MCP surface stay responsive even when the underlying workflow takes longer than a single request-response round trip.

This recipe now supports both invocation modes:

- **Synchronous call** — the tool starts a Durable Task orchestration and waits for completion before returning the result inline.
- **Task-augmented call** — the tool starts the orchestration and immediately returns MCP task metadata, using the orchestration instance ID as the `taskId`.

For this recipe, `recent_activity` advertises `execution.taskSupport = "optional"`, which means compatible MCP clients can choose whether to wait inline or treat the workflow as an MCP task.

### Task lifecycle mapping

| MCP task status | Durable Task orchestration status |
|---|---|
| `working` | `RUNNING`, `PENDING`, `SUSPENDED`, `CONTINUED_AS_NEW` |
| `completed` | `COMPLETED` |
| `failed` | `FAILED` |
| `cancelled` | `TERMINATED` |
| `input_required` | Orchestration waiting on `wait_for_external_event()` |

> **The `input_required` pattern:** An orchestration can pause durably by calling `ctx.wait_for_external_event("review_decision")` and setting a custom status with `awaiting_input: true`. The MCP server maps this to the `input_required` task state, and client input is forwarded via `raise_orchestration_event()`. This makes human approval a durable, resumable task state — not a special-case side channel.

### Supported MCP task operations

The MCP server implements the core task endpoints for durable orchestrations:

- `tools/call` with task augmentation schedules the orchestration and returns `CreateTaskResult`.
- `tasks/get` polls `get_orchestration_state()` and maps the orchestration runtime state to MCP task status.
- `tasks/result` waits for orchestration completion and returns the final tool payload.
- `tasks/cancel` calls `terminate_orchestration()` so the orchestration can stop durably.
- `tasks/list` queries orchestration instances to list active tasks.
- **Progress reporting** uses `set_custom_status(...)` to surface real-time status through the MCP task interface.

That gives any MCP-compatible host a standard async protocol while Durable Task provides the persistence, orchestration history, and recovery behavior underneath.

## Sample output

### Verifying the worker

```
$ python3 worker.py
Worker listening on http://localhost:8080 (taskhub=default).
Press Ctrl+C to stop.
```

The worker registers durable orchestrations that are invoked by the MCP server when tools are called from an MCP client.

> Note: This recipe does NOT have dashboard screenshots since it's triggered by MCP clients, not a standalone client.
