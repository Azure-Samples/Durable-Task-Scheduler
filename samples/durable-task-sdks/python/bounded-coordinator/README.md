# Bounded Coordinator

Python | Durable Task SDK

## Description of the Sample

This sample demonstrates the **bounded coordinator** pattern with the Azure Durable Task Scheduler using the Python SDK. A parent orchestration fans out child work in bounded batches, waits for all children to complete, and then resets its history via `continue_as_new` after each batch. This prevents the unbounded history growth that can occur with long-lived coordinator / message-pump orchestrations.

This is the Python counterpart to the .NET [BoundedCoordinator](../../dotnet/BoundedCoordinator/) sample.

In this sample:
1. The coordinator reads a **bounded batch** of items via the `get_next_batch` activity
2. It fans out child sub-orchestrations (`process_item_orchestrator`) for each item in the batch
3. It **waits for all children** to complete with `task.when_all`
4. If more work remains, it calls `continue_as_new` with compact carry-forward state
5. The orchestration restarts with a clean history and processes the next batch

This pattern is important because:
- Long-lived coordinators accumulate history events on every replay
- Without periodic resets, history can grow to tens of thousands of events
- Large histories cause increasingly expensive replays and can lead to persistence failures in backing stores

This pattern is useful for:
- Continuously draining a queue or work source in batches
- Long-running ingestion / ETL pipelines
- Any "message pump" style coordinator that must run indefinitely without history bloat

## Prerequisites

1. [Python 3.10+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample locally:

### Using the Emulator (Recommended)

The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. Pull the Docker Image for the Emulator:
   ```bash
   docker pull mcr.microsoft.com/dts/dts-emulator:latest
   ```

2. Run the Emulator:
   ```bash
   docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

Wait a few seconds for the container to be ready.

Note: The example code automatically uses the default emulator settings (endpoint: `http://localhost:8080`, taskhub: `default`). You don't need to set any environment variables.

### Using a Deployed Scheduler and Taskhub in Azure

Local development with a deployed scheduler:

1. Install the durable task scheduler CLI extension:

    ```bash
    az upgrade
    az extension add --name durabletask --allow-preview true
    ```

2. Create a resource group in a region where the Durable Task Scheduler is available:

    ```bash
    az provider show --namespace Microsoft.DurableTask --query "resourceTypes[?resourceType=='schedulers'].locations | [0]" --out table
    ```

    ```bash
    az group create --name my-resource-group --location <location>
    ```

3. Create a durable task scheduler resource:

    ```bash
    az durabletask scheduler create \
        --resource-group my-resource-group \
        --name my-scheduler \
        --ip-allowlist '["0.0.0.0/0"]' \
        --sku-name "Dedicated" \
        --sku-capacity 1 \
        --tags "{'myattribute':'myvalue'}"
    ```

4. Create a task hub within the scheduler resource:

    ```bash
    az durabletask taskhub create \
        --resource-group my-resource-group \
        --scheduler-name my-scheduler \
        --name "my-taskhub"
    ```

5. Grant the current user permission to connect to the `my-taskhub` task hub:

    ```bash
    subscriptionId=$(az account show --query "id" -o tsv)
    loggedInUser=$(az account show --query "user.name" -o tsv)

    az role assignment create \
        --assignee $loggedInUser \
        --role "Durable Task Data Contributor" \
        --scope "/subscriptions/$subscriptionId/resourceGroups/my-resource-group/providers/Microsoft.DurableTask/schedulers/my-scheduler/taskHubs/my-taskhub"
    ```

## How to Run the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. First, activate your Python virtual environment (if you're using one):
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows, use: venv\Scripts\activate
   ```

2. If you're using a deployed scheduler, set environment variables:
   ```bash
   export ENDPOINT=$(az durabletask scheduler show \
       --resource-group my-resource-group \
       --name my-scheduler \
       --query "properties.endpoint" \
       --output tsv)

   export TASKHUB="my-taskhub"
   ```

3. Install the required packages:
   ```bash
   pip install -r requirements.txt
   ```

4. Start the worker in a terminal:
   ```bash
   python worker.py
   ```

5. In a new terminal (with the virtual environment activated if applicable), run the client:
   > **Note:** Remember to set the environment variables again if you're using a deployed scheduler.

   ```bash
   python client.py
   ```

## Expected Output

### Client Output
```
Started coordinator_orchestrator, instance ID: <instance-id>

Completed with status: OrchestrationStatus.COMPLETED
Result: {"total_batches": 3, "completed": true}
```

### Worker Output
```
Coordinator batch 0, cursor=(start)
Processing batch of 5 items
Processing item item-1-1 for tenant tenant-1
Applying change for item item-1-1, tenant tenant-1
...
Batch 1 complete. 5 items processed.
Coordinator batch 1, cursor=cursor-1
...
Batch 3 complete. 5 items processed.
No more items to process. Coordinator completing.
```

## How It Works

```
coordinator_orchestrator (batch N)
  ├─ get_next_batch (bounded: up to MAX_ITEMS)
  ├─ fan out: process_item_orchestrator x batch size
  ├─ task.when_all(children)   ← wait for ALL children
  └─ continue_as_new(next state)   ← reset history, then batch N+1
```

The key is that the coordinator **waits for all children before resetting**, and
carries only compact state (`cursor`, `batch_number`) across the `continue_as_new`
boundary. Each restart begins with a fresh, small history.

### Anti-Pattern: Unbounded Coordinator

The following looks similar but never calls `continue_as_new`, so its history
grows without bound and replays get progressively slower:

```python
# BAD: this coordinator never resets its history
def coordinator(ctx, _):
    while True:
        item = yield ctx.wait_for_external_event("new-item")
        yield ctx.call_sub_orchestrator(process_item_orchestrator, input=item)
        # History grows by several events each iteration and is never reset
```

### Replay-safe logging

The orchestrators log through `ctx.create_replay_safe_logger(logger)` so log lines
are only emitted when the orchestrator is not replaying, avoiding duplicate logs.

## Viewing in the Dashboard

- **Emulator:** Navigate to http://localhost:8082 → select the "default" task hub
- **Azure:** Navigate to your Scheduler resource in the Azure Portal → Task Hub → Dashboard URL

Because the coordinator uses `continue_as_new`, you'll see a single coordinator
instance whose history stays small across batches, alongside the short-lived child
`process_item_orchestrator` instances.

## Related Samples

- [Eternal Orchestrations](../eternal-orchestrations/) - Single-orchestration `continue_as_new` loop
- [Sub-orchestrations](../sub-orchestrations/) - Composing parent/child orchestrations
- [Fan-out/Fan-in](../fan-out-fan-in/) - Parallel processing and aggregation

## Learn More

- [Eternal orchestrations](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-eternal-orchestrations)
- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler)
