# Async HTTP API (Go)

A `net/http` API accepts an operation request and starts `GoAsyncHTTPAPI`.
This workflow runs a simulated long-running activity on Durable Task Scheduler
(DTS). The API and worker run together. Workflow state is saved in DTS, not in
the HTTP server's memory.

The API returns **202 Accepted** while work continues in the background.
The **Location** header gives the status URL, and **Retry-After: 1** asks the
client to wait one second between checks. Keep checking that URL until it returns `200`.

## Prerequisites

- Go 1.25 or newer.
- A running DTS emulator, or an existing Azure scheduler/task hub.
- See [the shared Go README](../README.md) for emulator setup, dependencies, and
  live DTS identity/role configuration. This sample creates no Azure resources.

## Run one operation

From this sample directory:

```sh
go run .
# Optional overall deadline:
go run . -timeout 1m
```

From the Go module root, use `go run ./async-http-api`.
The demo starts a worker and a local HTTP server on an available port.
It submits one two-second operation, checks its Location URL, prints the result,
and shuts down. It is an example client, not a test suite.
The default runtime is two minutes. HTTP and worker shutdown have separate time
limits. The demo needs no extra runtime files and works from either directory.

Example output (IDs and timestamps vary):

```text
Accepted operation go-async-http-...; polling /api/operations/go-async-http-...
{
  "operation_id": "go-async-http-...",
  "status": "completed",
  "result": "Operation go-async-http-... completed successfully",
  "processed_at": ...
}
```

## Testing

```sh
go test .
# Opt in after configuring a running emulator or live DTS task hub:
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

Ordinary tests run offline. They check requests, HTTP errors, cancellation,
and allowed server addresses. `TestIntegration` starts the same worker and
handlers as the demo against real DTS. It checks `202`, `Location`,
`Retry-After`, and status polling. It compares HTTP and workflow results and
tests termination and `404` responses.

The integration test has a two-minute timeout and runs only when enabled.
Offline test replacements do not prove that durable execution works.
Go test output reports these checks separately from demo output.

## Code map / read order

| File | Responsibility |
|---|---|
| `main.go`, `app.go` | Command-line flags, worker registration, startup, and shutdown |
| `models.go`, `workflow.go` | Typed operation data, orchestration and activity |
| `http.go`, `server.go` | Routes, DTS access, and local HTTP server with time limits |
| `client.go` | One-job example client and JSON transport |
| `integration_test.go`, `main_test.go` | DTS integration tests and offline tests |

## Interactive server

```sh
go run . -serve -listen 127.0.0.1:8000 -timeout 10m
curl -i -X POST http://127.0.0.1:8000/api/start-operation \
  -H 'Content-Type: application/json' -d '{"processing_time":5}'
# Use the returned status_url / Location:
curl -i http://127.0.0.1:8000/api/operations/go-async-http-REPLACE
curl -i -X DELETE http://127.0.0.1:8000/api/operations/go-async-http-REPLACE
```

The server accepts only local addresses. `-timeout` and Ctrl+C stop the HTTP
server and worker. The default timeout is two minutes.
This teaching API has no authentication. Do not expose it to the public.

| Method | Route | Response |
|---|---|---|
| POST | `/api/start-operation` | `202`, operation ID, status URL, polling headers |
| GET | `/api/operations/{id}` | `202` while pending/running; `200` with Completed/Failed/Terminated/Canceled status when terminal |
| DELETE | `/api/operations/{id}` | `202` termination requested; `409` if already terminal |

`processing_time` defaults to 5 and must be an integer from 1–30 seconds.
Request bodies have a 4096-byte limit. Invalid or unknown fields return `400`.
A body that is too large returns `413`, and an unsupported content type returns
`415`. Missing instances or instances from other samples return `404`.
Backend errors return `502` or `504`.
A successful status lookup can report `status: "Failed"`. This does not mean
the workflow completed successfully.

DELETE requests termination. Termination stops orchestration progress;
**it cannot undo an activity's external side effects or guarantee interruption
of an already running activity**.
If the client disconnects, its HTTP wait ends, but the durable work continues.

## Configuration

| Environment variable | Default | Purpose |
|---|---|---|
| `DTS_CONNECTION_STRING` | unset | Full connection string; used instead of the settings below |
| `ENDPOINT` | `http://localhost:8080` | Emulator or live DTS endpoint |
| `TASKHUB` | `default` | Task hub |
| `DTS_AUTHENTICATION` | chosen automatically | `None` for local HTTP; `DefaultAzure` for live DTS |

The activity simulates work with a context-aware timer; it does not call a model
or an external operation. Connecting to live DTS changes where state is saved
and how the app signs in. The activity still uses simulated work.
Workers use automatic task filters and Go-specific stable task names.
