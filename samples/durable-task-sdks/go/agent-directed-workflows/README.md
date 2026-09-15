# Agent-directed workflows (Go)

Each chat session is a **durable entity**, `GoAgentDirectedChatAgent`, with persisted
conversation history, two protected receipt slots, and a bounded recovery cache.
DTS serializes its operations, including concurrent HTTP requests and resets. There is no process-memory
conversation store and no orchestration bridge.

The API supports messages, SSE, JSON, history, reset, and optional Azure OpenAI
tool calling. The default **mock mode is an explicitly labeled echo**, not an
intelligent agent.

## Prerequisites and run

- Go 1.25+ and a running DTS emulator or existing live task hub.
- [Shared Go README](../README.md): emulator connection, live DTS authentication,
  roles, and shared module setup.
- No Redis or model credentials are needed for the default demonstration.

From `samples/durable-task-sdks/go`:

```sh
go run ./agent-directed-workflows
go test -mod=readonly ./agent-directed-workflows
```

The bounded demo starts a worker and an actual loopback HTTP test server. It
asserts exact echo text, SSE chunk/done framing and headers, four persisted
conversation turns (including two concurrent requests), a committed reset, and
a fifth turn containing no old history. It also compares HTTP history with a
direct DTS entity read. The verification deadline is 65 seconds, plus bounded
worker shutdown.

Expected output:

```text
Chat mode: mock (mock is an echo, and the weather tool always uses synthetic data)
... "verified_turns": 5, "reset_verified": true ...
SAMPLE_OK agent-directed-workflows
```

The executable uses real DTS even in mock mode. Offline tests substitute a
test-only entity store and HTTP model server; those are not execution backends.

## Interactive API

```sh
go run ./agent-directed-workflows -serve -listen 127.0.0.1:5000 -timeout 10m
curl -N -X POST http://127.0.0.1:5000/chat/session1 \
  -H 'Content-Type: application/json' -d '{"message":"Weather in Seattle?"}'
curl -X POST 'http://127.0.0.1:5000/chat/session1?stream=false' \
  -H 'Content-Type: application/json' -d '{"message":"Hello again"}'
curl http://127.0.0.1:5000/chat/session1/history
curl -X POST http://127.0.0.1:5000/chat/session1/reset
```

| Method | Route | Contract |
|---|---|---|
| POST | `/chat/{session}` | SSE by default; `?stream=false` waits for committed JSON `{sessionId,message,mode}` |
| GET | `/chat/{session}/history` | `{sessionId,history,mode}` from the durable entity; missing session `404` |
| POST | `/chat/{session}/reset` | Waits for a durable reset acknowledgement, then `200` |
| GET | `/chat/{session}/requests/{request}` | Additional recovery endpoint: committed receipt, or `404` if queued/unknown/expired from retention |

Session IDs are 1–80 letters, digits, `_` or `-`. JSON bodies are capped at
4096 bytes and messages at 2048 bytes. Invalid JSON/unknown fields return `400`,
oversized bodies `413`, wrong media type `415`, and scheduler errors `502`/`504`.
Admission contention returns **`429` with `Retry-After: 1` before execution or SSE
headers**. It never runs the model and then reports admission backpressure.
Only explicit `stream=true` or `stream=false` values are accepted.
The server binds **only loopback**, shuts down on the configured deadline or
Ctrl+C, and has bounded request/read/write timeouts. It has no user authentication;
do not expose this teaching API publicly.

### Native SSE instead of Redis

```text
HTTP subscribes to a bounded, in-flight channel -> reserves a durable receipt slot
HTTP observes committed admission -> signals entity execution
entity streams model tokens -> local channel -> HTTP SSE chunks
entity commits history + protected receipt -> HTTP observes receipt -> SSE done
HTTP flushes the reply -> signals receipt acknowledgement -> slot can be reused
```

SSE events use the following format:

```text
data: {"type":"chunk","content":"Echo: "}

data: {"type":"done"}
```

Failures after streaming starts are `{"type":"error","content":"..."}` events
(the already-sent HTTP status stays `200`). Non-streaming failures use an HTTP
error code. Heartbeat comments keep idle streams active.

**Deliberate transport difference:** live tokens use bounded native Go channels,
not Redis pub/sub. These channels are transient transport only. If a worker is
in another process, or a slow reader loses chunks, HTTP reconstructs the remaining
reply from the durable receipt; that suffix streams **after commit**, not live.
No cross-node live-token distribution is claimed. A model retry can change a
provisional stream; in that case the API emits an error and directs the caller to
the committed history rather than falsely reporting success. `done` is never
emitted before the entity state is persisted.

### Durable admission and bounded receipt protection

`X-Chat-Request-ID` and `Content-Location` identify the durable receipt. Each
session has **two protected slots**, each capped at **16 KiB of serialized receipt
data**, including JSON escaping. HTTP uses generation-checked `reserve`
operations, observes a committed grant, and only then signals `message` or
`reset`. Competing/stale reservations cannot run a turn. Admission waits at most
five seconds before returning pre-execution `429`; a late reservation can hold a
slot temporarily, but cannot execute without the separate execution signal.

Active results are **never evicted by count or byte pressure**. They remain in
their slot until their owning HTTP handler has read the committed result,
successfully flushed the final JSON/SSE response, and signaled `ack`. This
survives an entity batch containing many operations: only admitted requests can
execute, and later operations cannot discard a result its HTTP owner still needs.
Generation checks fence delayed reserve, execution, and acknowledgement signals,
including across different HTTP/worker processes.

If the caller disconnects, delivery fails, or acknowledgement does not complete,
the admission lease bounds protection to **two minutes from reservation**,
longer than the 40-second original HTTP request lifetime. Unacknowledged receipts
remain recoverable within that lease, subject to normal backend availability.
An expired slot is reclaimed lazily by subsequent admissions; it needs no
background timer or unbounded per-request entity/orchestration store. Receipt GETs
are read-only: another reader cannot release a slot an active HTTP owner needs.

Only **acknowledged or lease-expired** receipts enter the evictable recovery cache
(at most 16 receipts / 32 KiB of serialized JSON). After acknowledgement, recovery
is best-effort within those caps, **not a guaranteed time window** or exactly-once
end-client delivery. A failed acknowledgement is logged; its outcome can be
ambiguous, so recovery may use either the protected slot or that cache. History
remains capped at 40 messages / 48 KiB and returns `409` when a reset is needed.
Reset clears conversation history, not other protected slots, cached receipts,
or scheduler audit history. Deploy the revised HTTP host and entity together;
direct SDK callers must use the reserve/execute/ack protocol too.

Offline stress regressions cover at least 17 concurrent short turns and four
near-8-KiB replies, including batched visibility and cross-process SSE fallback.
They require delivered success or explicit pre-execution backpressure, not a
completed turn whose caller loses its receipt. These use fault-injection
adapters, not an in-memory Go SDK backend; actual DTS stress is a separate check.

**Cancellation:** disconnecting cancels the HTTP wait, not an accepted entity
operation. Read history or the receipt URL instead of blindly resending a turn.
Queued operations expire after 35 seconds; an executing agent has a 25-second
budget. Entity operations remain serialized; reset uses the same admission and
receipt protection as messages. A receipt lease never extends the execution
deadline or makes a timed-out admission execute work.

## Optional real Azure OpenAI mode

```sh
export AZURE_OPENAI_ENDPOINT='https://YOUR-RESOURCE.openai.azure.com'
export AZURE_OPENAI_DEPLOYMENT='YOUR-CHAT-DEPLOYMENT'
# Optional: set AZURE_OPENAI_API_KEY securely, otherwise use DefaultAzureCredential.
go run ./agent-directed-workflows -serve -mode real -timeout 10m
```

Use a deployment supporting Chat Completions, streaming, and function tools.
The code uses the Azure Chat Completions REST API, with separate system/user/tool
messages. It accumulates streamed tool calls, executes the allowlisted
`get_weather` function, and calls the model again with tool results. The weather
tool, including in real mode, returns **synthetic 72°F/sunny example weather**;
it is not a live weather service.

All model I/O runs inside the **entity operation**, never an orchestrator.
The Go SDK's synchronous `EntityContext.Context()` supports context-bounded I/O.
The loop permits at most four model calls, eight tools, and an 8192-byte reply;
malformed/unknown tools are returned to the model as error data. Upstream,
authentication, stream, budget, and parsing errors fail the turn and do not
silently fall back to mock. Unsuccessful turns preserve prior history and
persist an error receipt. Model calls can repeat after a crash before commit:
neither model billing nor tool side effects are exactly-once.

Real mode requires `-serve`; the verification demo always requires mock mode,
including when testing live DTS. Real OpenAI calls are **not** claimed as tested.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `DTS_CONNECTION_STRING` | unset | Full shared connection string, takes precedence |
| `ENDPOINT` | `http://localhost:8080` | DTS endpoint |
| `TASKHUB` | `default` | DTS task hub |
| `DTS_AUTHENTICATION` | inferred | `None` for HTTP loopback; otherwise `DefaultAzure` |
| `CHAT_MODE` | `mock` | Default for `-mode`; credentials alone never enable real mode |
| `AZURE_OPENAI_ENDPOINT` | required in real mode | HTTPS Azure resource root, no path/query/userinfo |
| `AZURE_OPENAI_DEPLOYMENT` | required in real mode | Chat deployment name |
| `AZURE_OPENAI_API_VERSION` | `2024-10-21` | Chat API version |
| `AZURE_OPENAI_API_KEY` | unset | Optional API key; otherwise `DefaultAzureCredential` |
| `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_CLIENT_SECRET` | unset | Optional standard environment-credential inputs (or use Azure CLI / managed identity) |

Azure resource hosts are validated against documented Azure OpenAI/AI Services
domain suffixes; redirects are disabled to protect credentials. Credentials are
never persisted in entity state. Do not put secrets in chat messages: messages
and receipts are persisted. Worker task filters and entity names are Go-specific.
