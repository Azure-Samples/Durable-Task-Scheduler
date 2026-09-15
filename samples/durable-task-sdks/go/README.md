# Durable Task SDK samples for Go

Runnable Go counterparts to all [Python samples](../python/), using
[`microsoft/durabletask-go`](https://github.com/microsoft/durabletask-go)
**v1.0.0-beta.1**. This beta targets Durable Task Scheduler directly; it is not
the older Go SDK's embedded SQLite/PostgreSQL backend. Go is supported here as a
self-hosted Durable Task SDK, not as an Azure Functions language.

## Prerequisites

- **Go 1.25 or later**.
- Docker or a compatible container runtime for the DTS emulator.
- For Azure: an existing scheduler/task hub and an identity with the **Durable
  Task Data Contributor** role on the task hub or a containing scope.

The samples share one `go.mod` and pinned `go.sum`. Run commands from this Go
directory unless a sample README says otherwise.

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

Each sample starts its worker and client together, submits its demonstration,
checks the results, and shuts down. Successful verification ends with
`SAMPLE_OK <sample-name>`; failures return a nonzero exit status. Instances use
unique IDs and can be inspected in the [dashboard](http://localhost:8082).
Business activities such as payment, shipment, and device updates are
illustrative simulations, not production integrations.

Commands are bounded by `-timeout` (default `2m`). Use, for example,
`go run ./function-chaining -timeout 3m` on a high-latency connection.
The HTTP/agent samples also document their interactive server modes.

## Samples

| Sample | What it demonstrates |
|---|---|
| [Function chaining](function-chaining/) | Sequential activities and typed results |
| [Fan-out/fan-in](fan-out-fan-in/) | Parallel durable activities and aggregation |
| [Human interaction](human-interaction/) | Approval events, rejection, and durable timeout |
| [Monitoring](monitoring/) | Repeated checks with durable timers |
| [Eternal orchestrations](eternal-orchestrations/) | Bounded demonstration of `ContinueAsNew` |
| [Sub-orchestrations](sub-orchestrations/) | Composing child workflows |
| [Bounded coordinator](bounded-coordinator/) | Processing batches across fresh execution histories |
| [Saga](saga/) | Compensating actions after a failure |
| [Async HTTP API](async-http-api/) | HTTP 202 responses and status polling |
| [Entities](entities/) | Durable state, calls, signals, and scheduled signals |
| [Versioning](versioning/) | Version-aware workflow behavior and routing |
| [Work item filtering](work-item-filtering/) | Routing registered work to specialized workers |
| [Orchestration management](orchestration-management/) | Queries, restart, suspension, termination, and scoped cleanup |
| [Scheduled tasks](scheduled-tasks/) | Recurring schedules and their lifecycle |
| [Large payload](large-payload/) | Blob-backed payload externalization and verified round trips |
| [History export](history-export/) | Exporting terminal histories to Blob Storage |
| [OpenTelemetry tracing](opentelemetry-tracing/) | Caller/activity trace-context propagation and custom spans |
| [Agent-directed workflows](agent-directed-workflows/) | Entity-backed conversations and HTTP/SSE interaction |
| [arXiv research agent](arXiv_research_agent/) | Durable research workflows with fixture and external-provider modes |
| [Testing](testing/) | Offline business-logic tests and real DTS integration tests |

## Connect to Azure DTS

Authenticate locally with `az login` and use an existing **dedicated Go test
hub**. In particular, recurring schedules do not define a shared system-entity
state contract with other SDKs. History-export scans should not run over
unrelated workloads.

```bash
export DTS_ENDPOINT="$(az durabletask scheduler show \
  --resource-group <resource-group> --name <scheduler-name> \
  --subscription <subscription-id> --query properties.endpoint -o tsv)"

export DTS_CONNECTION_STRING="Endpoint=$DTS_ENDPOINT;TaskHub=<go-task-hub>;Authentication=AzureCLI"
go run ./function-chaining
```

Use `Authentication=DefaultAzure` for `DefaultAzureCredential`, including
managed identity or other supported Azure identity sources. Never put access
tokens or credentials in the repository. The connection string contains an
endpoint, task hub, and authentication choice, not an account key.

### Configuration

| Variable | Behavior |
|---|---|
| `DTS_CONNECTION_STRING` | Complete SDK connection string; takes precedence over the variables below |
| `ENDPOINT` | Scheduler endpoint; defaults to `http://localhost:8080` |
| `TASKHUB` | Task hub name; defaults to `default` |
| `DTS_AUTHENTICATION` | `None`, `DefaultAzure`, or `AzureCLI`; defaults to `None` only for a loopback HTTP endpoint, otherwise `DefaultAzure` |

The default connection is
`Endpoint=http://localhost:8080;TaskHub=default;Authentication=None`.
For an emulator on another host, explicitly select `Authentication=None`.
Do not use plaintext HTTP with Azure credentials.

## Build and test

```bash
go build ./...
go vet ./...
go test ./...
```

Normal tests require neither Azure nor an emulator. The catalog test compares
the Go suite to the Python directories so a new Python sample cannot silently
lose Go coverage.

The repository's [sample-build workflow](../../../.github/workflows/build-samples.yml)
also runs the executable suite against job-owned DTS and Azurite containers,
with fixture/mock AI modes and no live Azure credentials.

The Go beta has **no public in-memory orchestration test backend**. The
[testing sample](testing/) uses a local adapter to test the same business logic
offline; only integration runs exercise the real durable engine and replay.

### Verify every sample on either backend

The storage samples require a Blob endpoint. For a local test, start Azurite in
addition to DTS:

```bash
docker run -d --rm --name go-azurite \
  -p 127.0.0.1:10000:10000 mcr.microsoft.com/azure-storage/azurite:latest \
  azurite-blob --blobHost 0.0.0.0 --skipApiVersionCheck

HISTORY_EXPORT_ISOLATED_TASKHUB=1 DTS_SAMPLES_E2E=1 \
  go test -v -count=1 -timeout 30m ./e2e
```

`HISTORY_EXPORT_ISOLATED_TASKHUB=1` is a required acknowledgement for **both
emulator and Azure** export runs: the task hub must be isolated from unrelated
workloads and export workers. It does not create or isolate a task hub. The
export sample also guards the allowed instance IDs before reading histories.

Set `DTS_CONNECTION_STRING` to the Azure connection above and repeat the same
command for live DTS. The runner builds and executes **every sample program**,
checks its exit status and verification marker, and includes its assertion
output in the test log. It runs sequentially to avoid competing system workers.
To rerun one sample, use `-run 'TestSamples/function-chaining$'`.

**Verification boundaries:** the AI samples explicitly use fixtures/echo mode
by default, and the storage samples can use Azurite even when DTS is in Azure.
Those runs verify real DTS orchestration and worker-side integrations, not
live OpenAI/arXiv responses or Azure-hosted Blob Storage. See each sample's
README to configure and test those external services separately.

OpenTelemetry has a similar ownership boundary: DTS owns durable-operation
spans; Go propagates their trace context and emits the application's custom
spans. Follow the [tracing README](opentelemetry-tracing/) for collector setup.

Tests use their own IDs. Recurring/eternal demonstrations are bounded or stopped
explicitly. Completed and intentionally failed instances may remain for
dashboard inspection; use a dedicated task hub and delete that test hub after
testing rather than purging a shared hub.

## Learn more

- [Go SDK README and release notes](https://github.com/microsoft/durabletask-go)
- [Go API reference](https://pkg.go.dev/github.com/microsoft/durabletask-go)
- [DTS documentation](https://aka.ms/dts-documentation)
- [Repository sample catalog](../../README.md)
