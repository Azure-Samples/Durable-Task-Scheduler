# History export

Go | Durable Task SDK

Run five small square-number workflows, then archive their terminal histories
with the SDK's `exporthistory` extension. The demo shows the export destination
and job status, then deletes its own finite export job. Downloading and validating
the gzip JSONL archive is intentionally left to the integration tests.

## Prerequisites and isolation

- Go 1.25+ and the [shared emulator/live DTS setup](../README.md).
- **An isolated task hub, no other export workers, and no unrelated completions
  during the export window.** This applies to both emulator and live DTS.
- Azurite at `127.0.0.1:10000`, or an existing Azure Blob account.

The released SDK filters exports by completion window and terminal status, not
instance prefix, name, or tags. Its export system registrations are also shared
and unversioned. A unique destination is **not** source isolation. The required
acknowledgement below confirms that you supplied an isolated hub; it does not
create one.

The production ownership guards reject any listing page containing an unowned
ID, any unowned metadata/history read, and any write outside this run's
container/prefix. They do not silently skip unrelated instances.

## Run

From this directory, after confirming isolation:

```bash
export HISTORY_EXPORT_ISOLATED_TASKHUB=1
go run . -timeout 3m
```

From the shared Go module root, use `go run ./history-export -timeout 3m`.
Use `DTS_CONNECTION_STRING` or the shared `ENDPOINT`/`TASKHUB` settings for your
isolated live hub. The program does not provision task hubs or storage accounts.

| Environment | Blob destination |
| --- | --- |
| Neither Blob variable set | Public Azurite development account |
| `AZURE_STORAGE_CONNECTION_STRING` | Existing account; `UseDevelopmentStorage=true` is explicitly expanded |
| `AZURE_STORAGE_BLOB_ENDPOINT` | `https://<account>.blob.core.windows.net` with `DefaultAzureCredential` |

Set only one Blob variable. An Azure identity needs Blob read/write and container
creation permissions, such as Storage Blob Data Contributor. Only loopback HTTP
is allowed. The [large-payload compose file](../large-payload/docker-compose.yml)
can start Azurite if it is not already available.
**Live DTS plus Azurite exercises worker-side export storage, not Azure Blob.**

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
| [main.go](main.go) | Entrypoint and shared timeout |
| [workflow.go](workflow.go), [activities.go](activities.go) | Workflows whose histories are exported |
| [client.go](client.go) | Create and await a finite export job, display status |
| [worker.go](worker.go) | Register the real SDK export feature and start its worker |
| [sources.go](sources.go) | Owned source IDs, seeding, index visibility, and export window |
| [storage.go](storage.go), [ownership.go](ownership.go) | Blob setup and fail-closed privacy boundaries |
| [lifecycle.go](lifecycle.go) | Bounded job deletion and cancellation-safe worker shutdown |
| [integration_test.go](integration_test.go), [verify_test.go](verify_test.go) | Archive validation and active-job cancellation |

The window starts at the earliest source creation time rounded down to a second
and ends at the next second after the latest completion. This preserves valid
entries whose index timestamps differ from fine-grained metadata. All five owned
IDs must be listable before export begins; the broader window does not relax the
ownership guard.

## Testing

Offline tests:

```bash
go test -mod=readonly .
```

With isolation acknowledged and DTS/Blob storage configured:

```bash
DTS_SAMPLES_E2E=1 go test -run '^TestIntegration$' -v .
```

The integration test uses the production workers and workflows. It checks exact
square results, completed export status and counters, job-scoped listing, five
downloaded gzip JSONL blobs, deterministic names, metadata/schema, pinned execution
IDs, activity correlation, and terminal history results.

It also pauses a real export through a **test-only** storage wrapper, cancels the
scenario, and checks that the worker survives to delete the job and generation.
The CLI no longer reads `HISTORY_EXPORT_PAUSE_BEFORE_WRITE` or emits test-stage
markers. Offline lifecycle and ownership regression tests remain enabled without
services. Test cases are sequential and separate their coarse time windows from
previous completed control operations.

## Cleanup and limits

After any job creation attempt, cleanup has a fresh **30-second deadline** for
Delete and absence confirmation. The worker stays alive because Delete itself
needs durable execution. Host shutdown then has up to 20 seconds before the
worker lifetime is canceled. Connections still honor the original scenario
context. Original, cleanup, and shutdown failures produce a nonzero exit.

Only this job's captured generation is deleted. Source histories, Blob containers,
and completed SDK control-operation histories remain for inspection; no broad
purge is performed. Service/network failures can prevent bounded cleanup and are
reported with the job ID. Export is preview functionality; continuous schedules
and mixed export worker versions are outside this demo.

[Released export API and limitations](https://github.com/microsoft/durabletask-go/blob/v1.0.0-beta.1/exporthistory/README.md).
