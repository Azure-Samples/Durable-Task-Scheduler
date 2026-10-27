# Large payload externalization (Go)

The orchestration passes record data through two activities using
`payload.AzureBlobStore`. The client sends a small batch and a 2.1 MB batch.
The SDK stores large inputs and outputs in Blob Storage and loads them when
needed. The orchestration code has no Blob Storage code.

## Prerequisites

- The [shared Go README](../README.md) setup.
- Azurite at `127.0.0.1:10000`, or an existing Azure Blob Storage account. The
  [compose file](docker-compose.yml) starts **only Azurite**.

## Run

From this directory:

```bash
docker compose up -d   # only if Azurite is not already running
go run . -timeout 3m
```

Or, from the Go samples directory: `go run ./large-payload -timeout 3m`.

Blob settings are separate from the scheduler connection:

| Environment | Storage |
| --- | --- |
| Neither variable set | Azurite's [public development account](https://github.com/Azure/Azurite#default-storage-account) |
| `AZURE_STORAGE_CONNECTION_STRING` | Connection string for an existing account; the sample expands `UseDevelopmentStorage=true` into the Azurite settings |
| `AZURE_STORAGE_BLOB_ENDPOINT` | `https://<account>.blob.core.windows.net`, authenticated with `DefaultAzureCredential` |

Set only one variable. For Azure, use HTTPS and an identity with a role such as
Storage Blob Data Contributor. The sample does not create accounts or assign
roles. Using Azure DTS with Azurite does not test Azure Blob Storage.

Example output:

```text
Payload container: go-large-payload-...
go-large-payload-...: completed with 10 records (70 bytes)
go-large-payload-...: completed with 300000 records (2100000 bytes)
```

## How it works

- The SDK moves a payload to Blob Storage when it reaches **64 KiB**.
- Encoded payloads have a **4 MiB** limit.
- The sample lowers the gRPC message limit from the SDK default of 64 MiB to
  **128 KiB**, so the large batch cannot fit in one message.
- Each run adds a random ID to its task names. This stops workers on the same
  hub from taking another run's work.

## Code map

| File | Purpose |
| --- | --- |
| [main.go](main.go) | Command-line entry point |
| [workflow.go](workflow.go) | Echoes the payload, then processes its records |
| [activities.go](activities.go) | Echo and summary activities |
| [client.go](client.go) | Sends the two batches and prints the results |
| [worker.go](worker.go) | Registers task names and sets up storage |
| [storage.go](storage.go) | Azurite or Azure Blob authentication |
| [integration_test.go](integration_test.go), [verify_test.go](verify_test.go) | Backend tests and storage checks |

## Tests

```bash
go test -mod=readonly .
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The integration test runs two workers on the same hub. It checks that the
returned data matches the input (SHA-256), that the small batch creates no
blobs, and that the large batch creates valid gzip blobs.

## Cleanup

Completed instances and their Blob containers are kept. Do not delete blobs
while you keep histories that need them. To clean up, remove only the printed
instance IDs and container. `docker compose down -v` removes the data of the
private Azurite container; do not use it on shared storage.

See the [payload APIs](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta3/api/large_payload.go)
and the [Azure Blob implementation](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta3/payload/azure_blob.go).
