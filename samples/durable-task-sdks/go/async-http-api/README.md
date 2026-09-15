# Async HTTP API (Go)

A `net/http` API accepts a typed request and schedules
`GoAsyncHTTPAPI`, which runs a simulated long-running activity on Durable Task
Scheduler (DTS). The API process and worker run together. No state is kept in an
HTTP-server map.

The API implements the asynchronous HTTP protocol: **202 Accepted**, **Location**, and
**Retry-After: 1**. Poll the relative Location URL until it returns `200`.

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
The demo starts a worker and an ordinary loopback HTTP server on an ephemeral
port, submits one two-second operation, polls its Location URL, prints the
result, and shuts down. It is an example client, not a test suite. The shared
default runtime is two minutes; HTTP and worker shutdown are bounded separately.
No runtime files or working-directory-specific paths are needed.

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

Ordinary tests are offline and check request parsing, HTTP errors, cancellation,
and listener restrictions. `TestIntegration` starts the production worker and
handlers against real DTS, verifies 202/Location/Retry-After and pending polling,
compares HTTP output with durable output, and tests termination and `404`.
The integration test uses the shared two-minute test context and skips unless
explicitly enabled. Test doubles do not prove durable execution; Go test results
report verification separately from the demo.

## Code map / read order

| File | Responsibility |
|---|---|
| `main.go`, `app.go` | CLI flags, worker registration and lifetime |
| `models.go`, `workflow.go` | Typed operation data, orchestration and activity |
| `http.go`, `server.go` | Routes, backend adapter and bounded loopback server |
| `client.go` | One-job example client and JSON transport |
| `integration_test.go`, `main_test.go` | Real-backend verification and offline cases |

## Interactive server

```sh
go run . -serve -listen 127.0.0.1:8000 -timeout 10m
curl -i -X POST http://127.0.0.1:8000/api/start-operation \
  -H 'Content-Type: application/json' -d '{"processing_time":5}'
# Use the returned status_url / Location:
curl -i http://127.0.0.1:8000/api/operations/go-async-http-REPLACE
curl -i -X DELETE http://127.0.0.1:8000/api/operations/go-async-http-REPLACE
```

Only loopback addresses are accepted; `-timeout` and Ctrl+C shut down the HTTP
server and worker. The shared default timeout is two minutes. This unauthenticated
teaching API is not a public production endpoint.

| Method | Route | Response |
|---|---|---|
| POST | `/api/start-operation` | `202`, operation ID, status URL, polling headers |
| GET | `/api/operations/{id}` | `202` while pending/running; `200` with Completed/Failed/Terminated/Canceled status when terminal |
| DELETE | `/api/operations/{id}` | `202` termination requested; `409` if already terminal |

`processing_time` defaults to 5 and must be an integer from 1–30 seconds.
Bodies are limited to 4096 bytes; malformed/unknown fields return `400`,
oversized bodies `413`, unsupported media types `415`, missing or foreign sample
instances `404`, and backend failures `502`/`504`. A failed orchestration is a
successful status lookup with `status: "Failed"`, not a completed result.

DELETE requests termination. Termination stops orchestration progress;
**it cannot undo an activity's external side effects or guarantee interruption
of an already running activity**.
Client disconnection cancels the HTTP wait, not durable work.

## Configuration

| Environment variable | Default | Purpose |
|---|---|---|
| `DTS_CONNECTION_STRING` | unset | Shared helper's full connection string; takes precedence |
| `ENDPOINT` | `http://localhost:8080` | Emulator or live DTS endpoint |
| `TASKHUB` | `default` | Task hub |
| `DTS_AUTHENTICATION` | inferred | `None` for HTTP loopback; `DefaultAzure` for live DTS |

The activity simulates work with a context-aware timer; it does not call a model
or an external operation. Live DTS changes persistence/authentication, not that
simulation.
Workers use automatic task filters and Go-specific stable task names.
