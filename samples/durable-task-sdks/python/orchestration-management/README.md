# Orchestration Management

Python | Durable Task SDK

## Description of the Sample

This sample demonstrates orchestration lifecycle management operations with the Azure Durable Task Scheduler using the Python SDK. It covers restarting completed orchestrations, querying orchestration instances by filter, and batch purging old orchestrations.

In this sample:
1. Three orchestrations are scheduled and run to completion
2. A completed orchestration is restarted with the same instance ID
3. Another is restarted with a new instance ID
4. Orchestrations are queried by creation time and status
5. Completed orchestrations are batch-purged

This pattern is useful for:
- Re-running failed or completed workflows with the same original input
- Cleaning up old orchestration history to manage storage
- Querying orchestration status for monitoring dashboards
- Implementing retry-from-scratch logic in management tools

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
=== Step 1: Schedule orchestrations ===
  Completed: <id-1> -> {"batch_id": "batch-1", "items_processed": 10, "status": "success"}
  Completed: <id-2> -> {"batch_id": "batch-2", "items_processed": 20, "status": "success"}
  Completed: <id-3> -> {"batch_id": "batch-3", "items_processed": 30, "status": "success"}

=== Step 2: Restart orchestration (same instance ID) ===
  Restarted <id-1> -> new execution ID: <id-1>
  Restarted orchestration completed: {"batch_id": "batch-1", "items_processed": 10, "status": "success"}

=== Step 3: Restart orchestration (new instance ID) ===
  Restarted <id-2> with new ID: <new-id>
  New orchestration completed: {"batch_id": "batch-2", "items_processed": 20, "status": "success"}

=== Step 4: Query orchestrations ===
  Found 5 completed orchestration(s) since 2025-01-01T00:00:00+00:00

=== Step 5: Batch purge completed orchestrations ===
  Purged 5 orchestration(s)
  Remaining completed orchestrations: 0

Done!
```

## Code Walkthrough

### Restarting Orchestrations

Restart re-runs a completed orchestration with its original input:

```python
# Restart with the same instance ID (replaces the old execution)
restarted_id = client.restart_orchestration(instance_id)

# Restart with a new instance ID (keeps the old execution)
new_id = client.restart_orchestration(instance_id, restart_with_new_instance_id=True)
```

- **Same ID:** The restarted orchestration reuses the original instance ID. Useful for retrying a workflow in-place.
- **New ID:** A new instance ID is generated. Useful when you want to keep the history of the original execution.

### Querying Orchestrations

Query instances by creation time and status:

```python
query = durable_client.OrchestrationQuery(
    created_time_from=start_time,
    runtime_status=[durable_client.OrchestrationStatus.COMPLETED],
)
states = client.get_all_orchestration_states(query)
```

### Batch Purging

Remove completed orchestrations by filter criteria:

```python
result = client.purge_orchestrations_by(
    created_time_from=start_time,
    runtime_status=[durable_client.OrchestrationStatus.COMPLETED],
)
print(f"Purged {result.deleted_instance_count} orchestration(s)")
```

## Viewing in the Dashboard

- **Emulator:** Navigate to http://localhost:8082 → select the "default" task hub
- **Azure:** Navigate to your Scheduler resource in the Azure Portal → Task Hub → Dashboard URL

## Related Samples

- [Function Chaining](../function-chaining/) - Basic sequential workflow pattern
- [Async HTTP API](../async-http-api/) - RESTful API with orchestration backend

## Learn More

- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler)
