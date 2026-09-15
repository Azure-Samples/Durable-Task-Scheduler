# Large payload externalization

Go | Durable Task SDK

Send record data through an orchestration and two activities using
`payload.AzureBlobStore`. The client sends a small batch and a 2.1 MB batch,
receives the processed data, and prints a summary. The SDK transparently stores
large inputs and outputs in Blob storage; the workflow contains no Blob code.

## Prerequisites

- Go 1.25+ and the [shared emulator/live DTS setup](../README.md).
- Azurite at `127.0.0.1:10000`, or an existing Azure Blob account.
  The optional [compose file](docker-compose.yml) starts **only Azurite**.

## Run

From this directory:

```bash
go run . -timeout 3m
```

From the shared Go module root, use `go run ./large-payload -timeout 3m`.
If Azurite is not already running, `docker compose up -d` from this directory
starts it. Do not start another copy on an occupied port.

Blob configuration is independent of the scheduler connection:

| Environment | Storage |
| --- | --- |
| Neither variable set | Azurite's [public development account](https://github.com/Azure/Azurite#default-storage-account) |
| `AZURE_STORAGE_CONNECTION_STRING` | Existing account connection string; `UseDevelopmentStorage=true` is explicitly expanded |
| `AZURE_STORAGE_BLOB_ENDPOINT` | `https://<account>.blob.core.windows.net`, authenticated with `DefaultAzureCredential` |

Choose only one Blob variable. For Azure, the identity needs Blob data access and
container-creation permissions, such as Storage Blob Data Contributor. Only
loopback HTTP is permitted; use HTTPS for Azure. No account or role is provisioned.
**Live DTS with default Azurite exercises worker-side storage, not Azure Blob.**

Example output:

```text
Payload container: go-large-payload-...
go-large-payload-...: completed with 10 records (70 bytes)
go-large-payload-...: completed with 300000 records (2100000 bytes)
```

The demo is bounded and exits nonzero on workflow, storage, or shutdown errors.
It does not download blobs or run the test suite.

## Code map

| File | Responsibility |
| --- | --- |
| [main.go](main.go) | Entrypoint and shared timeout handling |
| [workflow.go](workflow.go) | Echo the payload, then process its records |
| [activities.go](activities.go) | Echo data and produce the record/byte summary |
| [client.go](client.go) | Submit the two batches and display their results |
| [worker.go](worker.go) | Register per-run task names and configure a shared client/worker payload store |
| [storage.go](storage.go) | Azurite or Azure Blob authentication |
| [integration_test.go](integration_test.go), [verify_test.go](verify_test.go) | Backend scenarios and detailed storage assertions |

The externalization threshold is **64 KiB**, the serialized payload limit is
**4 MiB**, and gRPC messages are capped at **128 KiB**. Thus the large batch cannot
travel inline. This deliberately lowers the SDK's usual 64 MiB gRPC bound.
Gzip and SDK integrity checks remain enabled.

Each invocation adds its random run/container ID to every registration,
orchestration scheduling name, and activity call name. Those names stay fixed
during replay. Concurrent runs on the same hub cannot execute each other's work
against different containers. This is a per-run teaching worker: new invocations
do not resume old in-flight instances. A shared production fleet instead needs
consistent task names and a compatible shared payload store.

## Testing

Offline tests:

```bash
go test -mod=readonly .
```

With the configured DTS backend and Blob storage available:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The integration test runs two workers concurrently on the same hub. Each uses
the production workflow and activities for small and large batches. Tests check
byte-for-byte and SHA-256 round trips, zero blobs for the small batch, actual
externalized blobs for the large batch, gzip decoding, and stored size/checksum
metadata. Offline tests also protect per-run registration isolation.

## Cleanup

Workers and clients stop automatically. Completed instances and their unique
Blob containers are retained for inspection: deleting blobs first would break
history hydration. Remove only the printed instance IDs/container when finished.
For a private compose instance, `docker compose down -v` removes its Azurite data;
do not use it to clean shared storage.

[Released payload APIs](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/api/large_payload.go)
and [Azure Blob implementation](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/payload/azure_blob.go).
