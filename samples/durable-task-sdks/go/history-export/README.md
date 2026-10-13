# History export

Go | Durable Task SDK

Run five small workflows that square numbers. Then use the SDK's `exporthistory`
extension to save their completed histories in Blob storage.
The demo shows the destination and job status, then deletes its export job.
The integration tests download and check the gzip JSONL files.

## Prerequisites and isolation

- Go 1.25+ and the [shared emulator/live DTS setup](../README.md).
- **A separate task hub with no other export workers. No unrelated workflows may
  complete during the export time range.** This applies to emulator and live DTS.
- Azurite at `127.0.0.1:10000`, or an existing Azure Blob account.

The SDK filters exports by completion time and final status.
It cannot filter by instance prefix, name, or tags. Its system tasks also use
shared names without versions. A unique destination **does not limit which
histories are read**. The setting below confirms that you have supplied a
separate hub. It does not create one.

Safety checks reject a result page if it contains an ID from another run.
They also block reads of other instances' metadata or history and writes outside
this run's container and prefix. These checks return errors instead of silently
skipping unrelated instances.

## Run

From this directory, after confirming isolation:

```bash
export HISTORY_EXPORT_ISOLATED_TASKHUB=1
go run . -timeout 3m
```

From the shared Go module root, use `go run ./history-export -timeout 3m`.
Use `DTS_CONNECTION_STRING` or the shared `ENDPOINT`/`TASKHUB` settings for your
separate live hub. The program does not create task hubs or storage accounts.

| Environment | Blob destination |
| --- | --- |
| Neither Blob variable set | Public Azurite development account |
| `AZURE_STORAGE_CONNECTION_STRING` | Existing account; the sample expands `UseDevelopmentStorage=true` into the Azurite settings |
| `AZURE_STORAGE_BLOB_ENDPOINT` | `https://<account>.blob.core.windows.net` with `DefaultAzureCredential` |

Set only one Blob variable. The Azure identity needs permission to read and
write Blob data and create containers. Storage Blob Data Contributor is one role
that provides this access. HTTP is allowed only on the local machine.
The [large-payload compose file](../large-payload/docker-compose.yml)
can start Azurite if it is not already available.
**Using live DTS with Azurite tests local export storage, not Azure-hosted Blob Storage.**

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

| File | Responsibility |
| --- | --- |
| [main.go](main.go) | Starts the command-line program and sets its timeout |
| [workflow.go](workflow.go), [activities.go](activities.go) | Workflows whose histories are exported |
| [client.go](client.go) | Create an export job, wait for it, and display its status |
| [worker.go](worker.go) | Register the real SDK export feature and start its worker |
| [sources.go](sources.go) | Create source instances, track their IDs, and choose the export time range |
| [storage.go](storage.go), [ownership.go](ownership.go) | Set up Blob storage and block access to unrelated data |
| [lifecycle.go](lifecycle.go) | Delete jobs and stop workers safely, with time limits |
| [integration_test.go](integration_test.go), [verify_test.go](verify_test.go) | Archive validation and active-job cancellation |

The time range starts at the earliest source creation time, rounded down to a
whole second. It ends at the next whole second after the last completion.
This allows for small differences between index and metadata timestamps.
All five source IDs must appear in the index before export starts.
The wider time range does not allow access to other instances.

## Testing

Offline tests:

```bash
go test -mod=readonly .
```

With isolation acknowledged and DTS/Blob storage configured:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The integration test uses the same workers and workflows as the demo.
It checks the square results, export status, counters, and job listing.
It downloads five gzip JSONL blobs and checks their names, format, metadata,
and execution IDs. Activity events must link to the correct results, and each
history must contain its final workflow result.

It also uses a **test-only** storage wrapper to pause a real export.
After cancelling the test scenario, it checks that the worker stays alive long
enough to delete the job and its execution.
The CLI no longer reads `HISTORY_EXPORT_PAUSE_BEFORE_WRITE` or emits test-stage
markers. Offline lifecycle and ownership regression tests remain enabled without
services. Test cases run one at a time. Their time ranges do not include control
operations completed by earlier cases.

## Cleanup and limits

After any attempt to create a job, cleanup has a separate **30-second deadline**.
It calls Delete and checks that the job no longer exists. The worker stays alive
because Delete itself needs durable execution. Host shutdown then has up to
20 seconds before the worker context is cancelled.
Connection setup still uses the original scenario context.
Scenario, cleanup, and shutdown errors return a nonzero exit status.

Cleanup deletes only the job execution tracked by this run.
Source histories, Blob containers, and completed SDK control histories remain
available. No hub-wide cleanup is performed.
Service or network failures may prevent cleanup within the time limit. Errors
include the job ID. Export is a preview feature. This demo does not cover
continuous export schedules or workers using different export versions.

[Released export API and limitations](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/exporthistory/README.md).
