# Large Payload Externalization

Python | Durable Task SDK

## Description of the Sample

This sample demonstrates large payload externalization with the Azure Durable Task Scheduler using the Python SDK. When orchestration inputs, activity outputs, or event data exceed a configurable size threshold, the SDK automatically offloads them to Azure Blob Storage and replaces them with compact reference tokens in gRPC messages.

In this sample:
1. A `generate_data` activity produces payloads of configurable size
2. A `process_data` activity receives and summarizes the data
3. The orchestrator runs twice — once with a small payload (stays inline) and once with a large payload (externalized to blob storage)
4. Externalization is completely transparent to the orchestration logic

This pattern is useful for:
- Workflows that process large documents, images, or datasets
- Avoiding gRPC message size limits when passing data between activities
- Keeping orchestration history compact while allowing large intermediate results

## Prerequisites

1. [Python 3.10+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator and Azurite)
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample locally:

### Using the Emulator (Recommended)

This sample provides a `docker-compose.yml` that starts both the DTS emulator and Azurite (Azure Storage emulator) together:

```bash
docker compose up -d
```

This starts:
- **DTS emulator** on ports 8080 (gRPC) and 8082 (dashboard)
- **Azurite** on ports 10000 (blob), 10001 (queue), and 10002 (table)

Wait a few seconds for both containers to be ready. To stop the services later:

```bash
docker compose down
```

Note: The example code automatically uses the default emulator settings (endpoint: `http://localhost:8080`, taskhub: `default`) and the Azurite connection string (`UseDevelopmentStorage=true`). You don't need to set any environment variables.

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
   export STORAGE_CONNECTION_STRING="DefaultEndpointsProtocol=https;AccountName=..."
   ```

3. Install the required packages:
   ```bash
   pip install -r requirements.txt
   ```

4. Start the worker in a terminal:
   ```bash
   python worker.py
   ```
   You should see output indicating the worker has started and registered the orchestration and activities.

5. In a new terminal (with the virtual environment activated if applicable), run the client:
   > **Note:** Remember to set the environment variables again if you're using a deployed scheduler.

   ```bash
   python client.py
   ```

## Expected Output

### Worker Output
```
Using taskhub: default
Using endpoint: http://localhost:8080
INFO:__main__:Starting Large Payload pattern worker...
INFO:__main__:Generating data with 10 records
INFO:__main__:Processed 10 records (70 bytes)
INFO:__main__:Generating data with 10000 records
INFO:__main__:Processed 10000 records (70000 bytes)
```

### Client Output
```
Using taskhub: default
Using endpoint: http://localhost:8080

--- Small payload (stays inline) ---
Result: "Processed 10 records (70 bytes)"

--- Large payload (externalized to blob storage) ---
Result: "Processed 10000 records (70000 bytes)"

Done!
```

Both orchestrations produce the same type of result. The difference is invisible to the application code — the SDK transparently externalizes the 70 KB payload to blob storage and retrieves it when needed.

## Code Walkthrough

### Payload Store Configuration

The `BlobPayloadStore` is configured with a `BlobPayloadStoreOptions` object:

```python
store = BlobPayloadStore(BlobPayloadStoreOptions(
    connection_string=storage_conn_str,
    threshold_bytes=1_024,  # Externalize payloads larger than 1 KB
))
```

Key options:
- **`connection_string`**: Azure Storage connection string (or `UseDevelopmentStorage=true` for Azurite)
- **`threshold_bytes`**: Payloads larger than this are externalized (default: 900 KB)
- **`max_stored_payload_bytes`**: Maximum payload size that can be stored (default: 10 MB)
- **`enable_compression`**: Whether to compress payloads with GZip before storing (default: `True`)
- **`container_name`**: Blob container name (default: `durabletask-payloads`)

### Passing the Store to Worker and Client

Both the worker and client must be configured with the same payload store:

```python
# Worker
with DurableTaskSchedulerWorker(..., payload_store=store) as worker:
    ...

# Client
client = DurableTaskSchedulerClient(..., payload_store=store)
```

## Viewing in the Dashboard

- **Emulator:** Navigate to http://localhost:8082 → select the "default" task hub
- **Azure:** Navigate to your Scheduler resource in the Azure Portal → Task Hub → Dashboard URL

## Related Samples

- [Function Chaining](../function-chaining/) - Basic sequential workflow pattern
- [Fan-Out/Fan-In](../fan-out-fan-in/) - Parallel processing pattern
- [Large Payload (.NET)](../../dotnet/LargePayload/) - Same pattern in .NET

## Learn More

- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler)
