# Orchestration management (Go)

## Description

The Go counterpart of [Python orchestration management](../../python/orchestration-management/)
demonstrates a bounded lifecycle using **only this invocation's instances**:

1. Schedule and complete three batches, producing 10, 20, and 30 processed items.
2. Restart the first using the same instance ID. Observe a **different execution
   ID** before waiting, so the old completed execution cannot create a false pass.
3. Restart the second using a new service-generated ID; verify the original
   execution and output are unchanged.
4. Suspend an event-gated batch, deliver an event while it is suspended, verify
   it stays suspended, then resume it and verify its output.
5. Terminate a second gated batch and verify `TERMINATED` and its reason.
6. Query all five completed instances using creation-time/status filters,
   owned-ID prefixes, and pagination.
7. Purge the **six exact owned IDs** (five completed, one terminated), verify
   each metadata lookup returns `api.ErrInstanceNotFound`, and verify both scoped
   queries are empty.

The Go/sample-specific registry is automatically filtered. Initial IDs have a
unique `go-management-*` prefix; the new-ID restart's returned ID is explicitly
tracked, since the service chooses it.

## Prerequisites

- Go 1.25.0 or later and the shared module's pinned
  `github.com/microsoft/durabletask-go v1.0.0-beta.1`.
- An existing emulator or Azure task hub with management data-plane access.
  Follow the [shared emulator/live authentication setup](../README.md).
  The demo creates no Azure resources.

## Run

From this directory:

```bash
go run .
```

The default scenario deadline is two minutes (`go run . -timeout 3m` changes it).
The management gate has a finite 45-second durable timeout, not an indefinite
timer. On failure the demo attempts to terminate only its tracked unfinished
instances using a fresh, bounded cleanup context before shutting down.

Offline tests:

```bash
go test -mod=readonly .
```

## Expected result

After all server states, outputs, restart identities, and deletions are verified:

```text
Completed batches: batch-1=10, batch-2=20, batch-3=30
Verified restart: same ID with new execution; new ID with original preserved
Verified SUSPENDED -> COMPLETED and RUNNING -> TERMINATED
Scoped query: 5 completed instances; exact-ID purge: 6 instances verified absent
SAMPLE_OK orchestration-management
```

An API acknowledgement is not considered a successful purge. If the target
cannot actually query, restart, suspend, terminate, or delete the owned instances,
the command fails and explains the failed verification; it does not log an
emulator limitation and print success. Unit tests specifically reject a
successful purge response whose metadata remains readable.

## Differences from Python

- Uses published Go `RestartInstance`, `QueryInstances`, and `PurgeInstances`
  APIs. It does **not** use `ListInstanceIDs`, which some emulator versions omit
  instances from.
- Python's time/status-wide batch purge is intentionally replaced by exact-ID,
  nonrecursive purging. No hub-wide query, query-and-delete sweep, or broad purge
  can touch unrelated work.
- Suspension, event buffering, resumption, termination, and strict result
  assertions extend the Python demo.
- One bounded process runs client and worker. A same-ID restart replaces an
  execution; it is not counted as an additional unique instance.
