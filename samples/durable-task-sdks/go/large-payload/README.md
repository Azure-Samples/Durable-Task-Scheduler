# Large payload externalization

Go | Durable Task SDK

Send record data through an orchestration and two activities using
`payload.AzureBlobStore`. The client sends a small batch and a 2.1 MB batch,
receives the processed data, and prints a summary. The SDK stores large inputs
and outputs in Blob storage and loads them when needed. This is called payload
externalization. The workflow does not need Blob storage code.

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
| `AZURE_STORAGE_CONNECTION_STRING` | Connection string for an existing account; the sample expands `UseDevelopmentStorage=true` into the Azurite settings |
| `AZURE_STORAGE_BLOB_ENDPOINT` | `https://<account>.blob.core.windows.net`, authenticated with `DefaultAzureCredential` |

Choose only one Blob variable. For Azure, the identity needs permission to read
and write Blob data and create containers. Storage Blob Data Contributor is one
role that provides this access. HTTP is allowed only on the local machine.
Use HTTPS for Azure. The sample does not create an account or assign roles.
**Using live DTS with Azurite tests local storage access, not Azure-hosted Blob Storage.**

Example output:

```text
Payload container: go-large-payload-...
go-large-payload-...: completed with 10 records (70 bytes)
go-large-payload-...: completed with 300000 records (2100000 bytes)
```

The demo has a time limit. It returns a nonzero exit status if a workflow,
storage operation, or shutdown fails. It does not download blobs or run the tests.

## Code map

| File | Responsibility |
| --- | --- |
| [main.go](main.go) | Starts the command-line program and sets its timeout |
| [workflow.go](workflow.go) | Echo the payload, then process its records |
| [activities.go](activities.go) | Echo data and produce the record/byte summary |
| [client.go](client.go) | Submit the two batches and display their results |
| [worker.go](worker.go) | Register this run's task names and set up storage for the client and worker |
| [storage.go](storage.go) | Azurite or Azure Blob authentication |
| [integration_test.go](integration_test.go), [verify_test.go](verify_test.go) | Backend test cases and detailed storage checks |

The SDK stores data in Blob storage once it reaches **64 KiB**. Encoded payloads
have a **4 MiB** limit, and gRPC messages have a **128 KiB** limit.
The large batch therefore cannot fit inside a gRPC message. The sample lowers
the usual SDK gRPC limit of 64 MiB to show this behavior.
Gzip compression and SDK data-integrity checks stay enabled.

Each run adds its random run/container ID to all task names. The same names are
used for registration, scheduling, and activity calls, including during replay.
This prevents workers on the same hub from taking another run's work and using
the wrong Blob container.

Each demo starts its own worker. A new demo run does not resume unfinished
instances from an older run. Production workers that share work need stable
task names and access to the same payload store.

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
that returned bytes and SHA-256 values match the original data.
The small batch must create no blobs. The large batch must create blobs with
valid gzip data, sizes, and checksums. Offline tests also check that runs use
separate task names.

## Cleanup

Workers and clients stop automatically. Completed instances and their Blob
containers remain available. Do not delete the blobs while keeping histories
that need them to load data. When finished, remove only the printed instance
IDs and their container.
For a private compose instance, `docker compose down -v` removes its Azurite data;
do not use it to clean shared storage.

[Released payload APIs](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/api/large_payload.go)
and [Azure Blob implementation](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/payload/azure_blob.go).
