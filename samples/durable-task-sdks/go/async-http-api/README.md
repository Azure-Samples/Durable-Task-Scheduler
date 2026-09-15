# Async HTTP API (Go)

A `net/http` counterpart to the Python sample: a typed HTTP request schedules
`GoAsyncHTTPAPI`, which runs a simulated long-running activity on Durable Task
Scheduler (DTS). The API process and worker run together. No state is kept in an
HTTP-server map.

Unlike the Python example's default `200` start response, this sample explicitly
implements the asynchronous HTTP protocol: **202 Accepted**, **Location**, and
**Retry-After: 1**. Poll the relative Location URL until it returns `200`.

## Prerequisites

- Go 1.25 or newer.
- A running DTS emulator, or an existing Azure scheduler/task hub.
- See [the shared Go README](../README.md) for emulator setup, dependencies, and
  live DTS identity/role configuration. This sample creates no Azure resources.

## Run the bounded demonstration

From `samples/durable-task-sdks/go`:

```sh
go run ./async-http-api
go test -mod=readonly ./async-http-api
```

The default run starts a worker and a real loopback HTTP test server, posts a
three-second job, observes pending responses, polls its result, and compares it
with the completed durable result. It also terminates a second job and checks
the terminal HTTP response and a missing-instance `404`. Verification has a
65-second deadline (plus bounded worker shutdown); it does not start an emulator.

Expected output includes an operation result:

```text
{
  "operation_id": "go-async-http-...",
  "status": "completed",
  "result": "Operation go-async-http-... completed successfully",
  "processed_at": ...
}
SAMPLE_OK async-http-api
```

`SAMPLE_OK` is only printed if all assertions and shutdown succeed.

## Interactive server

```sh
go run ./async-http-api -serve -listen 127.0.0.1:8000 -timeout 10m
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

DELETE is an additional Go convenience (the Python async sample has no DELETE
route). Termination stops orchestration progress; **it cannot undo an activity's
external side effects or guarantee interruption of an already running activity**.
Client disconnection cancels the HTTP wait, not durable work.

## Configuration

| Environment variable | Default | Purpose |
|---|---|---|
| `DTS_CONNECTION_STRING` | unset | Shared helper's full connection string; takes precedence |
| `ENDPOINT` | `http://localhost:8080` | Emulator or live DTS endpoint |
| `TASKHUB` | `default` | Task hub |
| `DTS_AUTHENTICATION` | inferred | `None` for HTTP loopback; `DefaultAzure` for live DTS |

There is no model or real external-operation mode: the activity deliberately
simulates work with a context-aware timer, exactly as the Python sample simulates
work with sleep. Live DTS changes persistence/authentication, not that simulation.
Workers use automatic task filters and Go-specific stable task names.
