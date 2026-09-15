# History export

Go | Durable Task SDK (preview export extension)

## Description

This counterpart to the [Python sample](../../python/history-export/) runs five
square-number orchestrations (`1, 4, 9, 16, 25`), exports their **terminal histories**
with the real `exporthistory` SDK extension, downloads the resulting gzip JSONL
blobs, and validates their contents. The command starts its worker and client
together and deletes its own completed export job before stopping.

## Prerequisites and isolation

- Go 1.25+ and the [shared emulator/live connection setup](../README.md).
- **An isolated task hub, with no other export workers and no unrelated workloads
  completing during the sample.** This applies to both emulator and live DTS.
- Azurite on `127.0.0.1:10000` or an existing Azure Blob account. From the shared
  Go module directory, the [large-payload compose file](../large-payload/docker-compose.yml)
  can start Azurite if it is not already running:

  ```bash
  docker compose -f large-payload/docker-compose.yml up -d
  ```

**Why isolation is mandatory:** in `v1.0.0-beta.1`, both
`JobCreationOptions` and `api.InstanceIDQuery` filter only by completion time and
terminal status. They have **no instance-ID, name, or tag filter**. A unique job or
blob prefix does not scope the histories being scanned. The SDK's built-in task
names (`ExportJob`, `ExportJobOrchestrator`, and its activities) are also shared,
unversioned system registrations. Do not mix .NET, Python, older Go, or another
copy of these export workers in the hub.

The sample additionally wraps the public `HistorySource` and `Store` interfaces
with an immutable allow-list of its five source IDs. It refuses an entire listing
page containing an unowned ID and rejects any direct unowned metadata/history
read or storage write. **It fails, rather than silently filtering/skipping other
instances or exporting unrelated user data.** This is defense in depth, not a
replacement for hub isolation.

## Run

From `samples/durable-task-sdks/go`, after confirming isolation:

```bash
export HISTORY_EXPORT_ISOLATED_TASKHUB=1
go run ./history-export
```

Use your existing isolated live task hub through `DTS_CONNECTION_STRING`, or
`ENDPOINT`/`TASKHUB`, as described in [the shared README](../README.md). The sample
does not create task hubs, accounts, or role assignments.

Blob configuration is independent:

| Variable | Behavior |
| --- | --- |
| Neither Blob variable set | Public Azurite development account |
| `AZURE_STORAGE_CONNECTION_STRING` | Connection string for an existing account; `UseDevelopmentStorage=true` is explicitly expanded |
| `AZURE_STORAGE_BLOB_ENDPOINT` | Account URL such as `https://<account>.blob.core.windows.net`, using `DefaultAzureCredential` |

Set only one Blob variable. Azure identities need Blob data read/write and
container-creation permissions, for example Storage Blob Data Contributor.
The sample creates a uniquely named container inside the selected account and
allows only that destination. Only loopback plaintext HTTP is permitted.

**Live DTS + default Azurite is worker-side export-storage validation, not an
Azure Blob integration test.** The worker performs the export writes; DTS does
not connect to your loopback Blob endpoint.

## Expected output and verification

```text
Completed go-history-export-source-...: 1 -> 1
...
Completed go-history-export-source-...: 5 -> 25
Export job: go-history-export-job-...; destination: go-history-export-.../...
EXPORT_JOB_CLEANUP job_id=go-history-export-job-...
EXPORT_JOB_CLEANED job_id=go-history-export-job-...
Verified 5 gzip JSONL blobs / ... history events; scanned=5 exported=5; job deleted
SAMPLE_OK history-export
```

The sample:

1. Checks all five source outputs and pins each source's execution ID.
2. Builds a completion-time window covering the sources' lifetimes, from the
   earliest creation through the next whole second after the last completion.
   A millisecond-tight window around completion metadata can omit a valid
   completion-index entry; the broader bounds do not weaken the owned-ID guard.
   It waits for all five IDs to be listable **before** creating the batch job.
   List visibility can lag completion; an empty export is never treated as success.
3. Uses pages of two instances, exercising real export-job pagination/checkpoints.
4. Requires both the durable job and its generation-specific orchestration to
   complete; checks exact batch scan/export counters and a job-ID-scoped listing.
5. Downloads only this run's prefix. For each blob it checks the deterministic
   filename, schema version, unpadded base64url instance/execution metadata,
   `application/gzip` with no `Content-Encoding`, a valid gzip stream, and JSON
   **on every line**.
6. Requires exactly one matching execution start, a correctly named/input square
   activity, a correlated activity result, and a successful terminal result.
   Missing, duplicate, corrupt, or unrelated histories fail the command.
7. Deletes **only this job ID** using the SDK and verifies it is no longer readable.

The default timeout is two minutes. A slow service/index can use
`go run ./history-export -timeout 5m`; failures and cleanup errors exit nonzero.
The SDK's per-instance and whole-page retry backoffs can exceed the default
timeout on persistent storage failures; fix the failure rather than treating a
timeout as a successful export.

## Cleanup and limitations

This is a finite **batch**, not a background export schedule. Job deletion cleans
its captured generation only. The five source histories and uniquely named Blob
container are retained for inspection; remove only their printed IDs/container
when finished. No broad purge or storage-container deletion is performed.

Worker lifetime is deliberately separate from the scenario's timeout/Ctrl-C
context. Connection setup still honors the scenario context, but a separately
cancellable worker remains alive for cleanup: the SDK's `JobClient.Delete` itself
schedules `ExecuteExportJobOperationOrchestrator`, which this isolated worker must
execute. On success, error, timeout, or Ctrl-C after job creation is attempted,
the sample gives **Delete plus absence verification a fresh, shared 30-second
deadline**. Only then does `Host.Close` drain/close the worker and client (up to
20 seconds), followed by cancellation of the worker lifetime.

`EXPORT_JOB_CLEANED` means Delete succeeded and the job is no longer readable.
An absent entity does not hide a failed generation purge. Original scenario,
cleanup, verification, and shutdown errors are preserved; a canceled scenario
does not print `SAMPLE_OK`. Service/network failures can still prevent bounded
cleanup, in which case the error includes this run's job ID. The SDK retains
completed control-operation histories; this sample does not broadly purge them.

### Opt-in active-job cancellation check

Set `HISTORY_EXPORT_PAUSE_BEFORE_WRITE=1` in addition to the isolation
acknowledgement. The first real export activity pauses before its Blob write and
prints exactly one stage signal:

```text
EXPORT_JOB_ACTIVE job_id=go-history-export-job-... paused_before_write=true
```

At that signal the job is genuinely executing and cannot finish its exports.
Send **one SIGINT to the sample process**, or allow its `-timeout` to expire.
The pause releases on scenario cancellation while the worker remains alive to
execute cleanup. Expect `EXPORT_JOB_CLEANUP`, then `EXPORT_JOB_CLEANED`, followed
by a **nonzero** canceled/deadline exit and no `SAMPLE_OK`. Allow up to 50 seconds
after cancellation for cleanup and host shutdown before force-killing a process.
For automated checks signal the compiled sample binary's PID, not a `go run`
wrapper. Do not set this opt-in flag during normal E2E success runs.

The preview job protocol is Go-specific. Its counters are cumulative processing
counts, not generally distinct-instance counts; this bounded, single-pass sample
can assert exactly five. Continuous exports, mixed worker versions, automatic
resource provisioning, and exporting shared production windows are not covered.

Unit tests run without services and test isolation guards, actual gzip/history
validation, cancellation/deadline cleanup ordering, error preservation, and the
opt-in pause signal:

```bash
go test -mod=readonly ./history-export
```

## API references

- [Released export feature and limitations](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/exporthistory/README.md)
- [Job options](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/exporthistory/options.go)
- [Streaming history storage](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/exporthistory/storage.go)
- [Upstream Go sample](https://github.com/microsoft/durabletask-go/tree/v1.0.0-beta.1/samples/exporthistory)
