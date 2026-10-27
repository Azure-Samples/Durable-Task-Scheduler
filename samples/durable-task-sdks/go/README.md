# Durable Task SDK samples for Go

These samples show how to build durable workflows in Go with
[`microsoft/durabletask-go`](https://github.com/microsoft/durabletask-go)
**v1.0.0-beta3**. The SDK connects directly to Durable Task Scheduler (DTS).
You host the Go app yourself. Go is not an Azure Functions language.

## Prerequisites

- Go **1.25 or later**.
- Docker, to run the DTS emulator.
- For Azure: an existing scheduler and task hub. Your identity needs the
  **Durable Task Data Contributor** role.

All samples share one `go.mod`. Run commands from this directory unless a
sample README says otherwise.

## Quickstart with the emulator

From the repository root:

```bash
docker run -d --rm --name go-dts-emulator \
  -p 127.0.0.1:8080:8080 -p 127.0.0.1:8082:8082 \
  mcr.microsoft.com/dts/dts-emulator:latest

cd samples/durable-task-sdks/go
go mod download
go run ./function-chaining
```

Each sample starts its worker and client in one process, runs a short demo,
prints the result, and exits. A failure returns a nonzero exit code. View
instances in the [dashboard](http://localhost:8082). Payment, shipping, and
similar business steps are simulations.

Every sample accepts `-timeout` (default `2m`), for example
`go run ./function-chaining -timeout 3m`.

## Code layout

Each sample README has a short code map. Most samples use this layout:

| File | Purpose |
|---|---|
| `main.go` | Command-line entry point |
| `workflow.go` | Orchestrations and their data types |
| `activities.go` | Activities called by the orchestrations |
| `worker.go` | Task registration and worker setup |
| `client.go` | Starts the demo and prints the result |
| `*_test.go` | Offline unit tests |
| `integration_test.go` | `TestIntegration` against real DTS (opt-in) |

## Samples

| Sample | What it shows |
|---|---|
| [Function chaining](function-chaining/) | Activities that run in order |
| [Fan-out/fan-in](fan-out-fan-in/) | Parallel activities and combined results |
| [Human interaction](human-interaction/) | Approval events with a durable timeout |
| [Monitoring](monitoring/) | Repeated checks with durable timers |
| [Eternal orchestrations](eternal-orchestrations/) | `ContinueAsNew` |
| [Sub-orchestrations](sub-orchestrations/) | Parent and child workflows |
| [Bounded coordinator](bounded-coordinator/) | Batches of child workflows with history reset |
| [Saga](saga/) | Undo earlier steps after a failure |
| [Async HTTP API](async-http-api/) | HTTP 202 responses and status polling |
| [Entities](entities/) | Durable state, calls, and signals |
| [Versioning](versioning/) | Versioned workflows |
| [Work item filtering](work-item-filtering/) | Route work to specific workers |
| [Orchestration management](orchestration-management/) | Query, restart, pause, stop, and purge instances |
| [Scheduled tasks](scheduled-tasks/) | Recurring schedules |
| [Large payload](large-payload/) | Store large inputs and outputs in Blob Storage |
| [History export](history-export/) | Export finished workflow histories to Blob Storage |
| [OpenTelemetry tracing](opentelemetry-tracing/) | Trace context and custom spans |
| [Testing](testing/) | Offline business-logic tests and DTS integration tests |

## Connect to Azure DTS

Sign in with `az login`. Use a **separate task hub** for the Go samples, because
Go schedules do not share a state format with other SDKs.

```bash
export DTS_ENDPOINT="$(az durabletask scheduler show \
  --resource-group <resource-group> --name <scheduler-name> \
  --subscription <subscription-id> --query properties.endpoint -o tsv)"

export DTS_CONNECTION_STRING="Endpoint=$DTS_ENDPOINT;TaskHub=<go-task-hub>;Authentication=AzureCLI"
go run ./function-chaining
```

Use `Authentication=DefaultAzure` to use `DefaultAzureCredential` (for example,
managed identity). The connection string has no secrets. Never commit tokens or
credentials.

| Variable | Behavior |
|---|---|
| `DTS_CONNECTION_STRING` | Full connection string; overrides the variables below |
| `ENDPOINT` | Scheduler endpoint; default `http://localhost:8080` |
| `TASKHUB` | Task hub name; default `default` |
| `DTS_AUTHENTICATION` | `None`, `DefaultAzure`, or `AzureCLI`; default is `None` for a loopback HTTP endpoint, otherwise `DefaultAzure` |

The default connection is
`Endpoint=http://localhost:8080;TaskHub=default;Authentication=None`.
For an emulator on another host, set `Authentication=None` explicitly. Do not
send Azure credentials over plain HTTP.

## Build and test

```bash
go build ./...
go vet ./...
go test ./...
```

These tests need no emulator or Azure. The Go beta SDK has **no public
in-memory test backend**, so offline tests do not check replay. The
[testing sample](testing/) shows how to test business logic offline.

To run one sample's integration test against a running backend:

```bash
DTS_SAMPLES_E2E=1 go test -v -run '^TestIntegration$' ./function-chaining
```

The [sample-build workflow](../../../.github/workflows/build-samples.yml) runs
all demos and integration tests in CI with its own DTS and Azurite containers.

### Verify every sample on either backend

The storage samples need a Blob endpoint. For local tests, also start Azurite:

```bash
docker run -d --rm --name go-azurite \
  -p 127.0.0.1:10000:10000 mcr.microsoft.com/azure-storage/azurite:latest \
  azurite-blob --blobHost 0.0.0.0 --skipApiVersionCheck

DTS_SAMPLES_E2E=1 \
  go test -v -count=1 -timeout 30m ./e2e
```

Configure the DTS emulator with both `default` and `historyexport` hubs
(`DTS_TASK_HUB_NAMES=default,historyexport`). For Azure, create a dedicated
`historyexport` hub. The runner uses that hub for history export and runs each
demo, then its `TestIntegration`, one sample at a time.
Skipped integration tests count as failures. To rerun one sample, add
`-run 'TestSamples/function-chaining$'`. For Azure, set `DTS_CONNECTION_STRING`
and run the same command.

These runs use real DTS. They do not test Azure-hosted Blob Storage when you use
Azurite. Completed instances stay in the dashboard. Use a separate test hub and
delete it when you finish. Do not clear a shared hub.

## Learn more

- [Go SDK README and release notes](https://github.com/microsoft/durabletask-go)
- [Go API reference](https://pkg.go.dev/github.com/microsoft/durabletask-go)
- [DTS documentation](https://aka.ms/dts-documentation)
- [Repository sample catalog](../../README.md)
