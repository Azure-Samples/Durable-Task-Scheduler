# History Export

Python | Durable Task SDK

## Description of the Sample

This sample demonstrates exporting the **event history of terminal orchestrations** to Azure Blob Storage using the `durabletask.extensions.history_export` extension with the Python SDK. An export job scans a time window of completed orchestrations and writes each instance's history to a blob container as gzipped JSONL.

This is the Python counterpart to the .NET [ExportHistoryWebApp](../../dotnet/ExportHistoryWebApp/) sample.

In this sample:
1. Five small orchestrations are scheduled and run to completion to populate history
2. An export job is created for the recent time window
3. The job is polled until it reaches a terminal status, then the result summary is printed

This pattern is useful for:
- Archiving orchestration history for compliance or auditing
- Moving completed-instance history out of the scheduler for long-term retention
- Feeding orchestration history into downstream analytics pipelines

## Prerequisites

1. [Python 3.10+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. An Azure Storage destination. For local development use [Azurite](https://learn.microsoft.com/azure/storage/common/storage-use-azurite), the Azure Storage emulator:
   ```bash
   npm install -g azurite
   azurite --silent --blobPort 10000
   ```
4. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

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

Note: The example code automatically uses the default emulator settings (endpoint: `http://localhost:8080`, taskhub: `default`). You don't need to set any environment variables for the scheduler.

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

## Configuring the Storage Destination

The export destination is configured with two environment variables (both optional):

- `STORAGE_CONNECTION_STRING` - Azure Storage connection string for the export destination. Defaults to `UseDevelopmentStorage=true` (Azurite).
- `EXPORT_CONTAINER` - Destination blob container name. Defaults to `history-export-sample`.

To export to a real Azure Storage account:
```bash
export STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=https;AccountName=...;AccountKey=...;EndpointSuffix=core.windows.net"
export EXPORT_CONTAINER="my-history-exports"
```

## How to Run the Sample

Once you have set up the emulator (or deployed scheduler) and Azurite, follow these steps:

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
   > **Note:** Remember to set the environment variables again if you're using a deployed scheduler. Set the same `STORAGE_CONNECTION_STRING` / `EXPORT_CONTAINER` values for both the worker and the client.

   ```bash
   python client.py
   ```

## Expected Output

### Client Output
```
=== Step 1: Seed sample orchestrations ===
  Completed: <id-1> -> 1
  Completed: <id-2> -> 4
  Completed: <id-3> -> 9
  Completed: <id-4> -> 16
  Completed: <id-5> -> 25

=== Step 2: Create export job ===
  job_id: <job-id>
  orchestrator_instance_id: export-job-<job-id>

=== Step 3: Wait for export job ===
  status:             completed
  scanned_instances:  5
  exported_instances: 5
  failed_instances:   0

Done!
```

After running, the gzipped JSONL history files appear in the `history-export-sample` container under the `sample-run/` prefix.

## Code Walkthrough

### Configuring the writer (worker and client)

The Azure Blob writer routes exported history to blob storage. The worker's
export activities perform the actual writes:

```python
from durabletask.extensions.history_export.azure_blob import (
    AzureBlobHistoryExportWriter,
    AzureBlobHistoryExportWriterOptions,
)

writer = AzureBlobHistoryExportWriter(
    AzureBlobHistoryExportWriterOptions(
        container_name=CONTAINER_NAME,
        connection_string=STORAGE_CONNECTION_STRING,
    )
)
```

### Registering the export feature on the worker

```python
from durabletask.extensions.history_export import ExportHistoryClient

export_client = ExportHistoryClient(client, writer)
export_client.register_worker(worker)  # entity + activities + orchestrator
```

### Creating and awaiting an export job (client)

```python
from durabletask.extensions.history_export import (
    ExportDestination, ExportFormat, ExportFormatKind,
    ExportJobCreationOptions, ExportMode,
)

desc = export_client.create_job(ExportJobCreationOptions(
    mode=ExportMode.BATCH,
    completed_time_from=now - timedelta(hours=1),
    completed_time_to=now + timedelta(hours=1),
    destination=ExportDestination(container=CONTAINER_NAME, prefix="sample-run"),
    format=ExportFormat(kind=ExportFormatKind.JSONL_GZIP),
    max_instances_per_batch=10,
))

final = export_client.wait_for_job(desc.job_id, timeout=120, poll_interval=0.5)
```

## Viewing in the Dashboard

- **Emulator:** Navigate to http://localhost:8082 → select the "default" task hub
- **Azure:** Navigate to your Scheduler resource in the Azure Portal → Task Hub → Dashboard URL

The export job itself runs as an orchestration (`export-job-<job-id>`) backed by a durable entity, so it is visible in the dashboard alongside the seeded sample orchestrations.

## Related Samples

- [Large Payload](../large-payload/) - Externalizing large payloads to Azure Blob Storage
- [Orchestration Management](../orchestration-management/) - Restart, query, and purge orchestrations
- [Entities](../entities/) - The durable entity feature that backs the export job

## Learn More

- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler)
