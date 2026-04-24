# Work Item Filtering — Split Activities Sample (Java)

This sample demonstrates **Work Item Filtering**, a feature that allows workers to declare which orchestrations, activities, and entities they can process. The Durable Task Scheduler (DTS) backend routes work items only to workers whose filters match, preventing workers from receiving work they cannot handle.

Before work item filtering, all orchestrations, activities, and entities were handed to any connected worker regardless of what it actually hosted. This caused errors (or silent hangs) when a worker received a work item it didn't implement — especially problematic in multi-service deployments, rolling upgrades, and microservice topologies. With filtering, each worker registers its task set; DTS creates per-filter queues and routes work items to matching workers. If no filter is specified, behavior falls back to the "generic queue" (all workers receive everything).

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Durable Task Scheduler (DTS)               │
│                                                             │
│  Orchestration queue ──► routed to Orchestrator Worker only │
│  ValidateOrder queue ──► routed to Validator Worker only    │
│  ShipOrder queue     ──► routed to Shipper Worker only      │
└────────────┬──────────────────┬──────────────────┬──────────┘
             │                  │                  │
     ┌───────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐
     │  Orchestrator  │  │  Validator    │  │  Shipper      │
     │  Worker        │  │  Worker       │  │  Worker       │
     │                │  │               │  │               │
     │ Registers:     │  │ Registers:    │  │ Registers:    │
     │ • OrderProc-   │  │ • Validate-   │  │ • ShipOrder   │
     │   essing-      │  │   Order       │  │               │
     │   Orchestration│  │               │  │               │
     └───────────────┘  └───────────────┘  └───────────────┘

     ┌───────────────┐
     │    Client      │
     │  (Driver)      │
     │                │
     │ Schedules new  │
     │ orchestrations │
     │ and prints     │
     │ results        │
     └───────────────┘
```

**Orchestrator Worker** runs orchestrations only — it has no activities registered.  
**Validator Worker** runs `ValidateOrder` only — it has no orchestrations or other activities.  
**Shipper Worker** runs `ShipOrder` only — same isolation.  
**Client** schedules orchestrations and polls for completion.

## The Orchestration

`OrderProcessingOrchestration` performs two sequential activity calls:

1. `ValidateOrder(orderId)` → routed to Validator Worker  
2. `ShipOrder(orderId)` → routed to Shipper Worker  

Returns a combined result string.

## Prerequisites

- [Java 21](https://adoptium.net/) (or later)
- [Docker](https://docs.docker.com/get-docker/) (for the DTS emulator)

## Running Locally

### 1. Start the DTS Emulator

```bash
docker pull mcr.microsoft.com/dts/dts-emulator:latest
docker run -d --name dts-emulator -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
```

The emulator dashboard is available at `http://localhost:8082`.

### 2. Build all projects

```bash
cd samples/scenarios/WorkItemFilteringSplitActivitiesJava
./gradlew build
```

### 3. Start the three workers (each in a separate terminal)

**Terminal 1 — Orchestrator Worker:**
```bash
./gradlew :orchestrator-worker:run
```

**Terminal 2 — Validator Worker (ValidateOrder activity):**
```bash
./gradlew :validator-worker:run
```

**Terminal 3 — Shipper Worker (ShipOrder activity):**
```bash
./gradlew :shipper-worker:run
```

### 4. Run the Client (in a fourth terminal)

```bash
./gradlew :client:run
```

## Expected Output

The client runs in a **continuous loop**, scheduling a batch of 3 orchestrations every 30 seconds for 10 minutes. This makes it easy to observe scaling behavior over time.

### Client terminal

```
10:30:01 INFO  i.d.samples.Client - === Work Item Filtering Demo — Client ===
10:30:01 INFO  i.d.samples.Client - Will schedule 3 orchestrations every 30s for 10 minutes.

10:30:01 INFO  i.d.samples.Client - --- Batch #1 at 2025-01-01T10:30:01Z ---
10:30:01 INFO  i.d.samples.Client - Scheduling orchestration with orderId='ORD-B001-001'...
10:30:01 INFO  i.d.samples.Client -   -> Scheduled with InstanceId=abc123
10:30:01 INFO  i.d.samples.Client - Scheduling orchestration with orderId='ORD-B001-002'...
10:30:01 INFO  i.d.samples.Client -   -> Scheduled with InstanceId=def456
10:30:01 INFO  i.d.samples.Client - Scheduling orchestration with orderId='ORD-B001-003'...
10:30:01 INFO  i.d.samples.Client -   -> Scheduled with InstanceId=ghi789
10:30:02 INFO  i.d.samples.Client - COMPLETED | InstanceId=abc123 | Output: Order 'ORD-B001-001' => Validation: [Order ORD-B001-001 is valid], Shipping: [Shipped with tracking TRACK-ORD-B001-001-4271]
...
10:30:02 INFO  i.d.samples.Client - Batch #1 results: 3 completed, 0 failed
```

### Orchestrator Worker terminal (orchestrations only — no activities)

```
10:30:02 INFO  i.d.samples.OrchestratorWorker - [Orchestrator] Orchestration | Name=OrderProcessingOrchestration | InstanceId=abc123 | Processing order 'ORD-B001-001'
10:30:02 INFO  i.d.samples.OrchestratorWorker - [Orchestrator] Orchestration | InstanceId=abc123 | Dispatching ValidateOrder to Validator Worker...
10:30:02 INFO  i.d.samples.OrchestratorWorker - [Orchestrator] Orchestration | InstanceId=abc123 | Dispatching ShipOrder to Shipper Worker...
10:30:02 INFO  i.d.samples.OrchestratorWorker - [Orchestrator] Orchestration | InstanceId=abc123 | Completed: Order 'ORD-B001-001' => Validation: [...], Shipping: [...]
```

### Validator Worker terminal (ValidateOrder only — no ShipOrder, no orchestrations)

```
10:30:02 INFO  i.d.samples.ValidatorWorker - [Validator] Activity | Name=ValidateOrder | InstanceId=abc123 | Validating order 'ORD-B001-001'...
10:30:02 INFO  i.d.samples.ValidatorWorker - [Validator] Activity | Name=ValidateOrder | InstanceId=abc123 | Result: Order ORD-B001-001 is valid
```

### Shipper Worker terminal (ShipOrder only — no ValidateOrder, no orchestrations)

```
10:30:02 INFO  i.d.samples.ShipperWorker - [Shipper] Activity | Name=ShipOrder | InstanceId=abc123 | Shipping order 'ORD-B001-001'...
10:30:02 INFO  i.d.samples.ShipperWorker - [Shipper] Activity | Name=ShipOrder | InstanceId=abc123 | Result: Shipped with tracking TRACK-ORD-B001-001-4271
```

**Key observation:** Each worker processes **only** its registered work item types. No cross-processing occurs.

## What to Try Next: Strict Routing Experiment

1. **Stop Shipper Worker** (Ctrl+C in Terminal 3).
2. Run the Client again to schedule new orchestrations.
3. Observe that:
   - Orchestrator Worker picks up and starts orchestrations.
   - Validator Worker completes `ValidateOrder` for each order.
   - `ShipOrder` work items **remain pending** — they are not delivered to Validator Worker or Orchestrator Worker.
   - The orchestrations stay in "Running" status, waiting for the `ShipOrder` activity to complete.
4. **Restart Shipper Worker** — the pending `ShipOrder` work items are immediately delivered and the orchestrations complete.

This demonstrates that filtering is strict: work items are routed only to workers with matching filters. There is no fallback to other workers.

## How It Works

Each worker process builds a `DurableTaskGrpcWorker` with only its own orchestrations or activities registered, then calls `useWorkItemFilters()` to auto-generate filters from the registry:

```java
DurableTaskGrpcWorker worker = DurableTaskSchedulerWorkerExtensions
        .createWorkerBuilder(connectionString)
        .addActivity(new TaskActivityFactory() { /* ... */ })
        .useWorkItemFilters() // auto-generate from registered tasks
        .build();
```

- Orchestrator Worker's filter: `orchestrations: [OrderProcessingOrchestration]`
- Validator Worker's filter: `activities: [ValidateOrder]`
- Shipper Worker's filter: `activities: [ShipOrder]`

DTS creates per-filter queues and routes each work item to the matching queue. If a filter list is empty for a given type (e.g., Validator Worker has no orchestration filter), that worker simply never receives work items of that type.

To supply explicit filters instead of auto-generating them, use `useWorkItemFilters(WorkItemFilter)`:

```java
WorkItemFilter filter = WorkItemFilter.newBuilder()
        .addOrchestration("OrderProcessingOrchestration")
        .addActivity("ValidateOrder")
        .build();

builder.useWorkItemFilters(filter);
```

## Deploying to Azure

This sample includes full infrastructure-as-code (Bicep) and an `azure.yaml` for one-command deployment via [Azure Developer CLI (`azd`)](https://learn.microsoft.com/azure/developer/azure-developer-cli/).

### What Gets Deployed

| Resource | Purpose |
|---|---|
| **Resource Group** | Contains all resources |
| **Durable Task Scheduler** (Consumption SKU) | Managed orchestration backend |
| **Task Hub** | Logical unit for orchestrations and work items |
| **Container Apps Environment** | Shared hosting environment with VNet integration |
| **Azure Container Registry** | Stores Docker images for each service |
| **User-Assigned Managed Identity** | Shared identity with DTS Worker/Client RBAC role |
| **4 Container Apps** | Client, Orchestrator Worker, Validator Worker, Shipper Worker |

### Deploy with `azd`

```bash
cd samples/scenarios/WorkItemFilteringSplitActivitiesJava
azd up
```

You'll be prompted for an environment name, subscription, and location. The deployment takes ~5 minutes.

### KEDA Scaling with DTS

Each worker Container App is configured with a **DTS-aware KEDA custom scale rule** (`azure-durabletask-scheduler`) that scales based on the **work item backlog** in the task hub. The key parameter is `workItemType`, which tells the scaler what kind of work to monitor:

| Container App | Service Name | `workItemType` | Scales on |
|---|---|---|---|
| **Client** | `client` | `Orchestration` | Pending orchestration work items |
| **Orchestrator Worker** | `orchestrator-worker` | `Orchestration` | Pending orchestration work items |
| **Validator Worker** | `validator-worker` | `Activity` | Pending activity work items |
| **Shipper Worker** | `shipper-worker` | `Activity` | Pending activity work items |

Workers scale from **0 to 10** replicas. When the client finishes its loop and no more work items arrive, workers scale back to zero.

### Manual Deployment (without `azd`)

Set the `ENDPOINT` and `TASKHUB` environment variables to point to your deployed scheduler:

```bash
export ENDPOINT="https://your-scheduler.westus2.durabletask.io"
export TASKHUB="your-taskhub-name"
```

The workers and client will automatically use `DefaultAzureCredential` for authentication. Make sure the identity running each process has the **Durable Task Scheduler Worker** / **Durable Task Scheduler Client** role on the scheduler resource.

## Project Structure

```
WorkItemFilteringSplitActivitiesJava/
├── build.gradle                   # Root Gradle build (shared dependencies)
├── settings.gradle                # Multi-project definition
├── README.md
├── azure.yaml                     # azd service definitions
├── .gitignore
├── infra/                         # Bicep infrastructure-as-code
│   ├── main.bicep                 # Top-level — resource group, DTS, container apps
│   ├── main.parameters.json
│   ├── abbreviations.json
│   ├── app/
│   │   ├── app.bicep              # Per-service container app (with KEDA scale rule)
│   │   ├── dts.bicep              # DTS scheduler + task hub
│   │   └── user-assigned-identity.bicep
│   └── core/
│       ├── host/                  # Container Apps Environment, Registry, App template
│       ├── networking/            # VNet
│       └── security/              # ACR pull role, DTS role assignments
├── shared/                        # Shared connection helper
│   ├── build.gradle
│   └── src/main/java/io/durabletask/samples/
│       └── ConnectionHelper.java
├── client/                        # Schedules orchestrations in a loop, prints results
│   ├── build.gradle
│   ├── Dockerfile
│   └── src/main/java/io/durabletask/samples/
│       └── Client.java
├── orchestrator-worker/           # Orchestrator Worker — runs orchestrations only
│   ├── build.gradle
│   ├── Dockerfile
│   └── src/main/java/io/durabletask/samples/
│       └── OrchestratorWorker.java
├── validator-worker/              # Validator Worker — runs ValidateOrder activity only
│   ├── build.gradle
│   ├── Dockerfile
│   └── src/main/java/io/durabletask/samples/
│       └── ValidatorWorker.java
└── shipper-worker/                # Shipper Worker — runs ShipOrder activity only
    ├── build.gradle
    ├── Dockerfile
    └── src/main/java/io/durabletask/samples/
        └── ShipperWorker.java
```

## Reference

- [Work Item Filtering PR (durabletask-java #275)](https://github.com/microsoft/durabletask-java/pull/275)
- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/durable-task-scheduler/)
- [Durable Task Java SDK](https://github.com/microsoft/durabletask-java)
