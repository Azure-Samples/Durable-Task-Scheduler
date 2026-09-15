# Durable Task SDK samples for Go

These samples show how to build Go workflows that save their progress. They use
[`microsoft/durabletask-go`](https://github.com/microsoft/durabletask-go)
**v1.0.0-beta.1**. This beta connects directly to Durable Task Scheduler.
It does not use the older Go SDK's built-in SQLite/PostgreSQL backend.
You host the Go application yourself. Go is not an Azure Functions language.

## Prerequisites

- **Go 1.25 or later**.
- Docker or a compatible container runtime for the DTS emulator.
- For Azure: an existing scheduler and task hub. Your identity needs the
  **Durable Task Data Contributor** role on the hub or a parent resource.

The samples share one `go.mod` and a `go.sum` file that records dependency checksums.
Run commands from this Go directory unless a sample README says otherwise.

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

Each sample starts its worker and client together, runs a short demonstration,
prints the result, and shuts down. Failures return a nonzero exit status.
Instances use unique IDs and can be inspected in the [dashboard](http://localhost:8082).
Detailed result and error checks run in test files, not in the demo.
Business activities such as payment, shipment, and device updates are
simulations. They do not use real payment, shipping, or device services.

The `-timeout` flag limits the runtime and defaults to `2m`. For a slow connection,
you can use `go run ./function-chaining -timeout 3m`.
The HTTP/agent samples also document their interactive server modes.

## Find the code

Start with the workflow or entity code to understand the pattern.
Each sample README includes a code map. The usual layout is:

| File | Responsibility |
|---|---|
| `main.go` | Starts the command-line program |
| `workflow.go` / `workflows.go` | Orchestrations and the data types they use |
| `activities.go` | Business operations called by workflows |
| `worker.go` | Task registration and worker setup |
| `client.go` | Start a demo and display its result |
| `*_test.go` | Unit tests, assertions, and verification helpers |
| `integration_test.go` | `TestIntegration` against real DTS, run only when enabled |

HTTP, entity, storage, and tracing samples use extra files named for those tasks.
Files stay in the same sample package, so you do not need to move between extra
package layers.

## Samples

| Sample | What it demonstrates |
|---|---|
| [Function chaining](function-chaining/) | Activities that run in order and pass results |
| [Fan-out/fan-in](fan-out-fan-in/) | Parallel activities and combined results |
| [Human interaction](human-interaction/) | Approval events, rejection, and durable timeout |
| [Monitoring](monitoring/) | Repeated checks with durable timers |
| [Eternal orchestrations](eternal-orchestrations/) | A short demo of `ContinueAsNew` |
| [Sub-orchestrations](sub-orchestrations/) | Parent and child workflows |
| [Bounded coordinator](bounded-coordinator/) | Processing batches across fresh execution histories |
| [Saga](saga/) | Steps that undo earlier work after a failure |
| [Async HTTP API](async-http-api/) | HTTP 202 responses and status polling |
| [Entities](entities/) | Durable state, calls, signals, and scheduled signals |
| [Versioning](versioning/) | Version-aware workflow behavior and routing |
| [Work item filtering](work-item-filtering/) | Routing registered work to specialized workers |
| [Orchestration management](orchestration-management/) | Find, restart, pause, stop, and clean up workflows |
| [Scheduled tasks](scheduled-tasks/) | Create and manage recurring schedules |
| [Large payload](large-payload/) | Store large data in Blob Storage and read it back |
| [History export](history-export/) | Save histories of finished workflows in Blob Storage |
| [OpenTelemetry tracing](opentelemetry-tracing/) | Trace context for clients and activities, plus custom spans |
| [arXiv research agent](arXiv_research_agent/) | Research workflows using sample data or real providers |
| [Testing](testing/) | Offline business-logic tests and real DTS integration tests |

## Connect to Azure DTS

Sign in with `az login` and use an existing **separate Go test hub**.
Go schedules do not share a state format with other SDKs.
Do not run history exports over unrelated workloads.

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
| `DTS_CONNECTION_STRING` | Full SDK connection string; used instead of the variables below |
| `ENDPOINT` | Scheduler endpoint; defaults to `http://localhost:8080` |
| `TASKHUB` | Task hub name; defaults to `default` |
| `DTS_AUTHENTICATION` | `None`, `DefaultAzure`, or `AzureCLI`; defaults to `None` only for a loopback HTTP endpoint, otherwise `DefaultAzure` |

The default connection is
`Endpoint=http://localhost:8080;TaskHub=default;Authentication=None`.
For an emulator on another host, explicitly select `Authentication=None`.
Do not send Azure credentials over unencrypted HTTP.

## Build and test

```bash
go build ./...
go vet ./...
go test ./...
```

Normal tests require neither Azure nor an emulator. The catalog test checks
sample coverage and checks that each sample has an entrypoint and
documentation.

The repository's [sample-build workflow](../../../.github/workflows/build-samples.yml)
also runs the demos and integration tests in its own DTS and Azurite containers.
The research agent uses sample data, and the job needs no live Azure credentials.

The Go beta has **no public in-memory orchestration test backend**. The
[testing sample](testing/) uses a local adapter to test business logic offline.
Only integration tests use the real durable engine and its replay behavior.

To run one sample's backend checks:

```bash
DTS_SAMPLES_E2E=1 go test -v -run '^TestIntegration$' ./function-chaining
```

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

For exports on **both emulator and Azure**, set `HISTORY_EXPORT_ISOLATED_TASKHUB=1`
only after checking that the hub is separate from unrelated work and export
workers. This setting does not create a hub or separate its data.
The sample also checks instance IDs before reading histories.

Set `DTS_CONNECTION_STRING` to the Azure connection above and repeat the same
command for live DTS. For each sample, the runner builds and runs the
**demonstration**, then builds a test binary and runs **`TestIntegration`**.
It checks the exit status and requires each integration test to run and pass
without skips. Results use normal Go test output, not markers in application
code. Both phases run one at a time so their workers do not compete.
To rerun one sample, use `-run 'TestSamples/function-chaining$'`.

**What the tests cover:** the research agent uses made-up sample data by
default, and the storage samples can use Azurite even when DTS is in Azure.
These tests use real DTS workflows and worker code. They do not verify
live OpenAI/arXiv responses or Azure-hosted Blob Storage. See each sample's
README to configure and test those external services separately.

DTS creates the OpenTelemetry spans for durable operations.
Go passes their trace context and creates the application's custom spans.
Follow the [tracing README](opentelemetry-tracing/) to set up a collector.

Tests use their own IDs. Recurring demos have limits or stop their work before
exiting. Completed and intentionally failed instances may remain in the dashboard.
Use a separate test hub and delete it after testing. Do not clear a shared hub.

## Learn more

- [Go SDK README and release notes](https://github.com/microsoft/durabletask-go)
- [Go API reference](https://pkg.go.dev/github.com/microsoft/durabletask-go)
- [DTS documentation](https://aka.ms/dts-documentation)
- [Repository sample catalog](../../README.md)
