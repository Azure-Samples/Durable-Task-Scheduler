# Work Item Filtering — Split Activities Sample (Python)

This sample demonstrates **Work Item Filtering**, a feature that allows workers to declare which orchestrations, activities, and entities they can process. The Durable Task Scheduler (DTS) backend routes work items only to workers whose filters match, preventing workers from receiving work they cannot handle.

Before work item filtering, all orchestrations, activities, and entities were handed to any connected worker regardless of what it actually hosted. This caused errors (or silent hangs) when a worker received a work item it didn't implement — especially problematic in multi-service deployments, rolling upgrades, and microservice topologies. With filtering, each worker registers its task set; DTS creates per-filter queues and routes work items to matching workers. If no filter is specified, a worker is eligible to receive any work item type or name, and DTS dispatches each work item to one eligible worker (it is not broadcast to every connected worker).

This is the Python version of the sample. Equivalent [.NET](../WorkItemFilteringSplitActivities/) and [Java](../WorkItemFilteringSplitActivitiesJava/) versions are also available.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Durable Task Scheduler (DTS)               │
│                                                             │
│  Orchestration queue ──► routed to Orchestrator Worker only │
│  validate_order queue ─► routed to Validator Worker only    │
│  ship_order queue    ──► routed to Shipper Worker only      │
└────────────┬──────────────────┬──────────────────┬──────────┘
             │                  │                  │
     ┌───────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐
     │  Orchestrator  │  │  Validator    │  │  Shipper      │
     │  Worker        │  │  Worker       │  │  Worker       │
     │                │  │               │  │               │
     │ Registers:     │  │ Registers:    │  │ Registers:    │
     │ • order_proc-  │  │ • validate_   │  │ • ship_order  │
     │   essing_      │  │   order       │  │               │
     │   orchestrator │  │               │  │               │
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
**Validator Worker** runs `validate_order` only — it has no orchestrations or other activities.
**Shipper Worker** runs `ship_order` only — same isolation.
**Client** schedules orchestrations and polls for completion.

## The Orchestration

`order_processing_orchestrator` performs two sequential activity calls:

1. `validate_order(order_id)` → routed to Validator Worker
2. `ship_order(order_id)` → routed to Shipper Worker

Returns a combined result string. The orchestrator calls each activity **by name**, so it doesn't need to import or register the activity functions — they run in separate worker processes.

## Prerequisites

- [Python 3.10+](https://www.python.org/downloads/)
- [Docker](https://docs.docker.com/get-docker/) (for the DTS emulator)
- [Azure Developer CLI (`azd`)](https://learn.microsoft.com/azure/developer/azure-developer-cli/) (for deploying to Azure)

## Running Locally

### Option A: One command

```bash
cd samples/scenarios/WorkItemFilteringSplitActivitiesPython
./run-local.sh
```

This starts the DTS emulator, creates a virtual environment, installs dependencies, and launches all three workers plus the client, tailing their logs. Press Ctrl+C to stop everything.

### Option B: Manual steps

#### 1. Start the DTS Emulator

```bash
docker pull mcr.microsoft.com/dts/dts-emulator:latest
docker run -d --name dts-emulator -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
```

The emulator dashboard is available at `http://localhost:8082`.

#### 2. Create a virtual environment and install dependencies

```bash
cd samples/scenarios/WorkItemFilteringSplitActivitiesPython
python -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate
pip install -r src/client/requirements.txt
```

All four services share the same two dependencies (`durabletask-azuremanaged` and `azure-identity`), so a single install covers everything.

#### 3. Start the three workers (each in a separate terminal)

**Terminal 1 — Orchestrator Worker:**
```bash
python src/orchestrator-worker/orchestrator_worker.py
```

**Terminal 2 — Validator Worker (validate_order activity):**
```bash
python src/validator-worker/validator_worker.py
```

**Terminal 3 — Shipper Worker (ship_order activity):**
```bash
python src/shipper-worker/shipper_worker.py
```

#### 4. Run the Client (in a fourth terminal)

```bash
python src/client/client.py
```

## Expected Output

The client runs in a **continuous loop**, scheduling a batch of 3 orchestrations every 30 seconds for 10 minutes. This makes it easy to observe scaling behavior over time.

### Client terminal

```
10:30:01 === Work Item Filtering Demo — Client ===
10:30:01 Will schedule 3 orchestrations every 30s for 10 minutes.

10:30:01 --- Batch #1 at 10:30:01 ---
10:30:01 Scheduling orchestration with orderId='ORD-B001-001'...
10:30:01   -> Scheduled with InstanceId=abc123
10:30:01 Scheduling orchestration with orderId='ORD-B001-002'...
10:30:01   -> Scheduled with InstanceId=def456
10:30:01 Scheduling orchestration with orderId='ORD-B001-003'...
10:30:01   -> Scheduled with InstanceId=ghi789
10:30:02 COMPLETED | InstanceId=abc123 | Output: "Order 'ORD-B001-001' => Validation: [Order ORD-B001-001 is valid], Shipping: [Shipped with tracking TRACK-ORD-B001-001-4271]"
10:30:02 Batch #1 results: 3 completed, 0 failed
10:30:02 Next batch in 30s (deadline in 10.0 min)
```

### Orchestrator Worker terminal (orchestrations only — no activities)

```
10:30:02 [Orchestrator] Orchestration | Name=order_processing_orchestrator | InstanceId=abc123 | Processing order 'ORD-B001-001'
10:30:02 [Orchestrator] Orchestration | InstanceId=abc123 | Dispatching validate_order to Validator Worker...
10:30:02 [Orchestrator] Orchestration | InstanceId=abc123 | Dispatching ship_order to Shipper Worker...
10:30:02 [Orchestrator] Orchestration | InstanceId=abc123 | Completed: Order 'ORD-B001-001' => Validation: [...], Shipping: [...]
```

### Validator Worker terminal (validate_order only — no ship_order, no orchestrations)

```
10:30:02 [Validator] Activity | Name=validate_order | InstanceId=abc123 | Validating order 'ORD-B001-001'...
10:30:02 [Validator] Activity | Name=validate_order | InstanceId=abc123 | Result: Order ORD-B001-001 is valid
```

### Shipper Worker terminal (ship_order only — no validate_order, no orchestrations)

```
10:30:02 [Shipper] Activity | Name=ship_order | InstanceId=abc123 | Shipping order 'ORD-B001-001'...
10:30:02 [Shipper] Activity | Name=ship_order | InstanceId=abc123 | Result: Shipped with tracking TRACK-ORD-B001-001-4271
```

**Key observation:** Each worker processes **only** its registered work item types. No cross-processing occurs.

## What to Try Next: Strict Routing Experiment

1. **Stop Shipper Worker** (Ctrl+C in Terminal 3).
2. Let the Client schedule new orchestrations.
3. Observe that:
   - Orchestrator Worker picks up and starts orchestrations.
   - Validator Worker completes `validate_order` for each order.
   - `ship_order` work items **remain pending** — they are not delivered to Validator Worker or Orchestrator Worker.
   - The orchestrations stay in "Running" status, waiting for the `ship_order` activity to complete.
4. **Restart Shipper Worker** — the pending `ship_order` work items are immediately delivered and the orchestrations complete.

This demonstrates that filtering is strict: work items are routed only to workers with matching filters. There is no fallback to other workers.

## How It Works

Each worker process registers its tasks with the `DurableTaskSchedulerWorker` and then calls `use_work_item_filters()`. The SDK automatically constructs **work item filters** from whatever is registered:

- Orchestrator Worker's filter: `orchestrations: [order_processing_orchestrator]`
- Validator Worker's filter: `activities: [validate_order]`
- Shipper Worker's filter: `activities: [ship_order]`

```python
# Orchestrator Worker — registers only the orchestrator
worker.add_orchestrator(order_processing_orchestrator)
worker.use_work_item_filters()  # Auto-generate filters from the registry
```

DTS creates per-filter queues and routes each work item to the matching queue. If a filter list is empty for a given type (e.g., Validator Worker has no orchestration filter), that worker simply never receives work items of that type.

For more control, you can pass explicit `WorkItemFilters` with optional version constraints:

```python
from durabletask import worker

worker.use_work_item_filters(worker.WorkItemFilters(
    activities=[
        worker.ActivityWorkItemFilter(name="validate_order"),
    ],
))
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
cd samples/scenarios/WorkItemFilteringSplitActivitiesPython
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

The scale rule metadata (from [app.bicep](infra/app/app.bicep)):

```bicep
scaleRuleType: 'azure-durabletask-scheduler'
scaleRuleMetadata: {
  endpoint: dtsEndpoint          // DTS scheduler URL
  maxConcurrentWorkItemsCount: '1'
  taskhubName: taskHubName
  workItemType: workItemType     // 'Orchestration' or 'Activity'
}
scaleRuleIdentity: userAssignedManagedIdentity.resourceId
```

- Workers scale from **0 to 10** replicas. When the client finishes its loop and no more work items arrive, workers scale back to zero.
- The `scaleRuleIdentity` uses the shared user-assigned managed identity to authenticate with DTS, so no connection strings or secrets are needed for scaling.
- `maxConcurrentWorkItemsCount: '1'` means KEDA will scale up one replica per pending work item, up to the max.

### Manual Deployment (without `azd`)

Set the `ENDPOINT` and `TASKHUB` environment variables to point to your deployed scheduler:

```bash
export ENDPOINT="https://your-scheduler.westus2.durabletask.io"
export TASKHUB="your-taskhub-name"
```

The workers and client will automatically use `DefaultAzureCredential` for authentication. Make sure the identity running each process has the **Durable Task Scheduler Worker** / **Durable Task Scheduler Client** role on the scheduler resource. When running inside Container Apps, the `AZURE_MANAGED_IDENTITY_CLIENT_ID` environment variable selects the shared user-assigned managed identity.

## Project Structure

```
WorkItemFilteringSplitActivitiesPython/
├── README.md
├── azure.yaml                     # azd service definitions
├── run-local.sh                   # Local run helper (emulator + all services)
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
└── src/
    ├── client/                    # Schedules orchestrations in a loop, prints results
    │   ├── client.py
    │   ├── requirements.txt
    │   └── Dockerfile
    ├── orchestrator-worker/       # Orchestrator Worker — runs orchestrations only
    │   ├── orchestrator_worker.py
    │   ├── requirements.txt
    │   └── Dockerfile
    ├── validator-worker/          # Validator Worker — runs validate_order activity only
    │   ├── validator_worker.py
    │   ├── requirements.txt
    │   └── Dockerfile
    └── shipper-worker/            # Shipper Worker — runs ship_order activity only
        ├── shipper_worker.py
        ├── requirements.txt
        └── Dockerfile
```

## Viewing in the Dashboard

- **Emulator:** Navigate to `http://localhost:8082` → select the "default" task hub.
- **Azure:** Navigate to your Scheduler resource in the Azure Portal → Task Hub → Dashboard URL.

## Reference

- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler)
- [Durable Task Python SDK](https://github.com/microsoft/durabletask-python)
