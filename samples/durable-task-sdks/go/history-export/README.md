# History export (Go)

The demo runs five small orchestrations that square numbers. Then it uses the
SDK's `exporthistory` extension (preview) to save their completed histories to
Blob Storage. It prints the destination and job status and deletes the export
job. Exported history blobs are kept.

## Prerequisites

- The [shared Go README](../README.md) setup.
- Azurite at `127.0.0.1:10000`, or an existing Azure Blob Storage account. The
  [large-payload compose file](../large-payload/docker-compose.yml) can start
  Azurite.
- **A separate task hub** with no unrelated orchestrations completing during
  the export time range. The export API filters by completion time and status,
  not by instance ID.

Use a task hub dedicated to this sample. The demo and integration test also
create their own source histories in that hub.

## Run

Set `DTS_CONNECTION_STRING` to the dedicated task hub, then run:

```bash
go run . -timeout 3m
```

Or, from the Go samples directory: `go run ./history-export -timeout 3m`.

| Environment | Blob destination |
| --- | --- |
| Neither Blob variable set | Public Azurite development account |
| `AZURE_STORAGE_CONNECTION_STRING` | Existing account; the sample expands `UseDevelopmentStorage=true` into the Azurite settings |
| `AZURE_STORAGE_BLOB_ENDPOINT` | `https://<account>.blob.core.windows.net` with `DefaultAzureCredential` |

Set only one Blob variable. The Azure identity needs a role such as Storage
Blob Data Contributor. HTTP is allowed only for localhost. Using Azure DTS
with Azurite does not test Azure Blob Storage.

Example output:

```text
Completed go-history-export-source-...: 1 -> 1
...
Completed go-history-export-source-...: 5 -> 25
Destination: go-history-export-.../go-history-export-job-.../
Export job go-history-export-job-...: Completed (5 histories exported)
Export job deleted; history blobs retained.
```

## Code map

| File | Purpose |
| --- | --- |
| [main.go](main.go) | Command-line entry point |
| [workflow.go](workflow.go), [activities.go](activities.go) | Orchestrations to export |
| [client.go](client.go) | Creates the export job and shows its status |
| [worker.go](worker.go) | Registers the export feature and starts the worker |
| [sources.go](sources.go) | Creates source instances and picks the time range |
| [storage.go](storage.go) | Configures Blob Storage |
| [integration_test.go](integration_test.go), [verify_test.go](verify_test.go) | Checks exports and job cancellation |

## Tests

```bash
go test -mod=readonly .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

Run the integration test only on a dedicated hub with Blob Storage configured.
It downloads the five gzip JSONL blobs and checks their content. It also
cancels an active export job and checks cleanup.

## Cleanup

The sample deletes its export job. It keeps source histories and Blob containers.

See the [export API and limitations](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta3/exporthistory/README.md).
