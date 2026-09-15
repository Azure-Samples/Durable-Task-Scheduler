# Large payload externalization

Go | Durable Task SDK

## Description

This sample generates `RECORD|` data, passes it between activities, and processes
it transparently using the Go SDK's `payload.AzureBlobStore`.

One command runs a filtered worker and a client, first with 10 records (70 bytes),
then with 300,000 records (2,100,000 bytes). It additionally sends the full expected
data as orchestration input and returns it as output, exercising **both client and
worker** externalization/hydration.

- Threshold: **64 KiB**; maximum serialized payload: **4 MiB**.
- Both directions of gRPC are deliberately capped at **128 KiB**, far below the
  large input/output. The SDK normally defaults to 64 MiB; this sample does **not**
  claim that 2 MiB exceeds that default.
- Azure Blob's SDK defaults are 256 KiB/10 MiB; its threshold cannot exceed 1 MiB.
- Gzip compression and the SDK's blob integrity checks remain enabled.

### Per-run worker isolation

Each invocation uses its random container/run ID as a suffix on **all three task
names**, including the client scheduling name and both activity call names. The
worker's automatic filters advertise only those names. Two invocations can
therefore run concurrently on the **same task hub and Azurite account** without
executing one another's work or writing payloads into the wrong container.

Separate containers alone are not enough: the `GenerateData` activity's small
inline input could otherwise be dispatched to either worker, even though its
large output must be resolved from the originating run's store.

Names are generated once before registration, captured in the workflow value,
and remain stable during replay. This is a **per-run teaching worker**, not a
shared fleet or a restart/resume tool: a new invocation gets new names and does
not resume a previous invocation's in-flight instances. A production fleet using
shared task names must share a compatible payload store/container configuration.
The sample does not broaden the Blob resolver's container allow-list.

## Prerequisites

- Go 1.25+ and an emulator or existing live DTS task hub:
  [shared connection/authentication setup](../README.md).
- Azurite listening on `127.0.0.1:10000`, or an existing Azure Blob account.
  The optional compose file starts **only Azurite**:

  ```bash
  docker compose -f large-payload/docker-compose.yml up -d
  ```

Run commands from `samples/durable-task-sdks/go`. Do not start a second Azurite if
the shared environment already has one.

## Run

```bash
go run ./large-payload
```

No Blob environment variables are needed locally. The code uses Azurite's
[public development account and connection string](https://github.com/Azure/Azurite#default-storage-account).
This is not an Azure account credential. `UseDevelopmentStorage=true` is expanded
explicitly because the Go Azure Blob client does not implement that shorthand.

For an existing Azure Blob account choose **one**:

```bash
# Supply a connection string securely through your environment.
export AZURE_STORAGE_CONNECTION_STRING='<your connection string>'
# OR use your already authenticated DefaultAzureCredential identity:
export AZURE_STORAGE_BLOB_ENDPOINT='https://<account>.blob.core.windows.net'
```

The identity needs Blob data read/write and container-creation permissions (for
example, Storage Blob Data Contributor). The sample creates a unique container
inside that account; it does not provision an account or change role assignments.
Only loopback HTTP storage is allowed; use HTTPS for Azure.

Storage and scheduler configuration are independent. **Live DTS with the default
Azurite destination validates worker-side Blob storage, not Azure Blob connectivity.**
The service stores references; this process's worker/client access the blobs.

## Expected output and checks

```text
Payload container: go-large-payload-... (retained for history hydration)
Instance: go-large-payload-... (10 records)
Verified 10 records / 70 bytes; SHA-256=...; stored blobs=0
Instance: go-large-payload-... (300000 records)
Verified 300000 records / 2100000 bytes; SHA-256=...; stored blobs=...
SAMPLE_OK large-payload
```

Success requires:

1. Both orchestrations complete, with exact content, record count, length, and
   SHA-256 round trips.
2. The small run produces **no** blobs.
3. The large run produces at least four blobs in its unique container.
4. Every blob is downloaded, decompressed, checked against the SDK's
   `durabletask_size`/`durabletask_sha256` metadata, and compared byte-for-byte with
   the generated record data. Orchestration success alone is not sufficient.

Failures exit nonzero, including shutdown errors. Use `-timeout 5m` on a slow
environment. Unit tests require no services and check that two runs have
disjoint orchestration/activity registrations and stable per-run names:

```bash
go test -mod=readonly ./large-payload
```

## Cleanup

The process stops its worker/client. It intentionally retains its two completed
orchestrations and the printed, uniquely named container: deleting payload blobs
while retaining histories would break future hydration. Remove only those
explicit instance IDs and that container when finished inspecting them. For local
storage, `docker compose -f large-payload/docker-compose.yml down -v` removes the
compose project's Azurite data; do not do this against a shared Azurite instance.

## API references

- [Azure Blob payload store](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/payload/azure_blob.go)
- [Public large-payload options and limits](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/api/large_payload.go)
- [Upstream Go sample](https://github.com/microsoft/durabletask-go/tree/v1.0.0-beta.1/samples/largepayloads)
