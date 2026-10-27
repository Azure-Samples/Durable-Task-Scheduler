# Async HTTP API (Go)

A `net/http` API starts the `GoAsyncHTTPAPI` orchestration, which runs a
simulated long-running activity. The API returns **202 Accepted** with a
**Location** header (the status URL) and **Retry-After: 1**. The client polls
the status URL until it returns `200`. Workflow state is stored in DTS, not in
the server's memory.

## Prerequisites

See the [shared Go README](../README.md) for Go, emulator, and Azure setup.

## Run one operation

From this directory:

```sh
go run .
# Optional overall deadline:
go run . -timeout 1m
```

Or, from the Go samples directory: `go run ./async-http-api`. The demo starts
a worker and a local HTTP server, submits one two-second operation, polls it,
and prints the result.

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

## Interactive server

```sh
go run . -serve -listen 127.0.0.1:8000 -timeout 10m
curl -i -X POST http://127.0.0.1:8000/api/start-operation \
  -H 'Content-Type: application/json' -d '{"processing_time":5}'
# Use the returned status_url / Location:
curl -i http://127.0.0.1:8000/api/operations/go-async-http-REPLACE
curl -i -X DELETE http://127.0.0.1:8000/api/operations/go-async-http-REPLACE
```

The server accepts only local addresses and has no authentication. Do not
expose it publicly.

| Method | Route | Response |
|---|---|---|
| POST | `/api/start-operation` | `202`, operation ID, status URL, polling headers |
| GET | `/api/operations/{id}` | `202` while running; `200` with Completed/Failed/Terminated/Canceled status when done |
| DELETE | `/api/operations/{id}` | `202` termination requested; `409` if already done |

`processing_time` defaults to 5 and must be 1–30 seconds. Bad input returns
`400`, a body over 4096 bytes returns `413`, and a wrong content type returns
`415`. Unknown instances return `404`. Backend errors return `502` or `504`.
A `200` response can still report `status: "Failed"`.

Termination stops the orchestration. It **cannot undo side effects of an
activity that is already running**.

## Code map

| File | Purpose |
|---|---|
| `main.go`, `app.go` | Flags, worker registration, startup, and shutdown |
| `models.go`, `workflow.go` | Operation data, orchestration, and activity |
| `http.go`, `server.go` | Routes and the local HTTP server |
| `client.go` | Example polling client |
| `integration_test.go`, `main_test.go` | Integration and offline tests |

## Tests

```sh
go test .
# Opt in after configuring a running emulator or live DTS task hub:
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

Offline tests check request handling and HTTP errors. `TestIntegration` uses
real DTS and checks `202`, `Location`, `Retry-After`, polling, termination,
and `404` responses.
