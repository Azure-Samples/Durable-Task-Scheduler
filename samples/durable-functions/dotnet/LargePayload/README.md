<!--
---
description: This sample demonstrates large payload support in .NET isolated Durable Functions with the Durable Task Scheduler.
page_type: sample
products:
- azure-functions
- durable-functions
- dts
- azure
- entra-id
urlFragment: large-payload-dotnet
languages:
- csharp
---
-->

# Large Payload Support - .NET Isolated Durable Functions

This sample demonstrates how to use the **large payload storage** feature with .NET isolated Durable Functions and the [Durable Task Scheduler](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler). When enabled, payloads that exceed a configurable threshold are automatically offloaded to Azure Blob Storage (compressed via gzip), keeping orchestration history lean while supporting arbitrarily large data.

The sample uses a **fan-out/fan-in** pattern: the orchestrator fans out to multiple activity functions, each generating a configurable-size payload (default 100 KB). The orchestrator then aggregates the results.

## How large payload storage works

The Durable Task Scheduler has a per-message size limit. When `largePayloadStorageEnabled` is set to `true` in `host.json`, any orchestration input, output, or activity result that exceeds `largePayloadStorageThresholdBytes` is:

1. Compressed with gzip
2. Uploaded to a blob container (`durabletask-payloads` by default) in the storage account configured via `AzureWebJobsStorage`
3. Replaced in the orchestration history with a small reference pointer

This happens transparently — no code changes are required.

## Prerequisites

- [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
- [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
- [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli)
- A [Durable Task Scheduler](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler) resource with a task hub
- An Azure Storage account (for large payload blob storage)
- [Docker](https://docs.docker.com/engine/install/) (optional, for local emulator)

## Project structure

```
LargePayload/
├── LargePayload.csproj             # Project file with NuGet package references
├── LargePayloadOrchestration.cs    # Orchestrator, activity, and HTTP trigger
├── Program.cs                      # Host builder setup
├── host.json                       # Host configuration with large payload settings
└── README.md
```

## Key configuration in host.json

```json
"durableTask": {
  "storageProvider": {
    "type": "azureManaged",
    "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING",
    "largePayloadStorageEnabled": true,
    "largePayloadStorageThresholdBytes": 10240
  },
  "hubName": "%TASKHUB_NAME%"
}
```

| Setting | Description | Default |
|---|---|---|
| `largePayloadStorageEnabled` | Enables large payload externalization to blob storage | `false` |
| `largePayloadStorageThresholdBytes` | Payloads larger than this (in bytes) are externalized | `10240` (10 KB) |

## NuGet packages

This sample uses the following published packages:

| Package | Version |
|---|---|
| `Microsoft.Azure.Functions.Worker.Extensions.DurableTask` | 1.14.1 |
| `Microsoft.Azure.Functions.Worker.Extensions.DurableTask.AzureManaged` | 1.3.0 |

The `AzureManaged` package includes `Microsoft.DurableTask.Extensions.AzureBlobPayloads` as a transitive dependency, which provides the blob externalization capability.

## Running locally with the emulator

1. Pull and run the Durable Task Scheduler emulator:

    ```bash
    docker run -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
    ```

    Port `8080` exposes the gRPC endpoint and `8082` exposes the monitoring dashboard.

2. Start Azurite (local storage emulator):

    ```bash
    azurite start
    ```

3. Create a `local.settings.json` file:

    ```json
    {
      "IsEncrypted": false,
      "Values": {
        "FUNCTIONS_WORKER_RUNTIME": "dotnet-isolated",
        "AzureWebJobsStorage": "UseDevelopmentStorage=true",
        "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=http://localhost:8080;Authentication=None",
        "TASKHUB_NAME": "default",
        "PAYLOAD_SIZE_KB": "100",
        "ACTIVITY_COUNT": "5"
      }
    }
    ```

4. Build and run the function app:

    ```bash
    func start
    ```

    Expected output:
    ```
    Hello: [GET] http://localhost:7071/api/Hello
    LargePayloadFanOutFanIn: orchestrationTrigger
    ProcessLargeData: activityTrigger
    StartLargePayload: [GET,POST] http://localhost:7071/api/StartLargePayload
    ```

5. Trigger the orchestration:

    ```bash
    curl -X POST http://localhost:7071/api/StartLargePayload
    ```

    The response includes a `statusQueryGetUri` you can poll to check orchestration progress.

## Deploy to Azure

### 1. Create Azure resources

If you don't already have them, create the required Azure resources:

```bash
# Create a resource group
az group create --name my-rg --location <location>

# Create a Durable Task Scheduler and task hub
az durabletask scheduler create --name my-scheduler --resource-group my-rg --location <location> --sku-name <sku-name>
az durabletask taskhub create --scheduler-name my-scheduler --resource-group my-rg --name my-taskhub

# Create a storage account
az storage account create --name mystorageaccount --resource-group my-rg --location <location> --sku Standard_LRS

# Create a function app (.NET 8 isolated, Linux)
az functionapp create \
  --name my-func-app \
  --resource-group my-rg \
  --storage-account mystorageaccount \
  --consumption-plan-location <location> \
  --runtime dotnet-isolated \
  --runtime-version 8.0 \
  --os-type Linux \
  --functions-version 4
```

### 2. Configure identity-based authentication

The Durable Task Scheduler **requires** identity-based authentication (managed identity). You can use either system-assigned or user-assigned managed identity.

#### Option A: System-assigned managed identity

```bash
# Enable system-assigned managed identity
az functionapp identity assign --name my-func-app --resource-group my-rg

# Get the principal ID
PRINCIPAL_ID=$(az functionapp identity show --name my-func-app --resource-group my-rg --query principalId -o tsv)

# Grant "Durable Task Data Contributor" role on the scheduler
SCHEDULER_ID=$(az durabletask scheduler show --name my-scheduler --resource-group my-rg --query id -o tsv)
az role assignment create --assignee $PRINCIPAL_ID --role "Durable Task Data Contributor" --scope $SCHEDULER_ID

# Grant "Storage Blob Data Contributor" role on the storage account (for large payload blobs)
STORAGE_ID=$(az storage account show --name mystorageaccount --resource-group my-rg --query id -o tsv)
az role assignment create --assignee $PRINCIPAL_ID --role "Storage Blob Data Contributor" --scope $STORAGE_ID
```

Configure app settings for system-assigned identity:

```bash
SCHEDULER_ENDPOINT=$(az durabletask scheduler show --name my-scheduler --resource-group my-rg --query endpoint -o tsv)

az functionapp config appsettings set --name my-func-app --resource-group my-rg --settings \
  "DURABLE_TASK_SCHEDULER_CONNECTION_STRING=Endpoint=${SCHEDULER_ENDPOINT};TaskHub=my-taskhub;Authentication=ManagedIdentity" \
  "AzureWebJobsStorage__accountName=mystorageaccount" \
  "FUNCTIONS_WORKER_RUNTIME=dotnet-isolated" \
  "TASKHUB_NAME=my-taskhub"
```

> **Note:** For system-assigned identity, you only need `AzureWebJobsStorage__accountName`. No `__credential` or `__clientId` is required — the SDK uses `DefaultAzureCredential` automatically.

#### Option B: User-assigned managed identity

```bash
# Create a user-assigned identity
az identity create --name my-identity --resource-group my-rg

IDENTITY_CLIENT_ID=$(az identity show --name my-identity --resource-group my-rg --query clientId -o tsv)
IDENTITY_PRINCIPAL_ID=$(az identity show --name my-identity --resource-group my-rg --query principalId -o tsv)
IDENTITY_ID=$(az identity show --name my-identity --resource-group my-rg --query id -o tsv)

# Assign the identity to the function app
az functionapp identity assign --name my-func-app --resource-group my-rg --identities $IDENTITY_ID

# Grant roles (same as above, using IDENTITY_PRINCIPAL_ID)
az role assignment create --assignee $IDENTITY_PRINCIPAL_ID --role "Durable Task Data Contributor" --scope $SCHEDULER_ID
az role assignment create --assignee $IDENTITY_PRINCIPAL_ID --role "Storage Blob Data Contributor" --scope $STORAGE_ID
```

Configure app settings for user-assigned identity:

```bash
az functionapp config appsettings set --name my-func-app --resource-group my-rg --settings \
  "DURABLE_TASK_SCHEDULER_CONNECTION_STRING=Endpoint=${SCHEDULER_ENDPOINT};TaskHub=my-taskhub;Authentication=ManagedIdentity;ClientId=${IDENTITY_CLIENT_ID}" \
  "AzureWebJobsStorage__accountName=mystorageaccount" \
  "AzureWebJobsStorage__credential=managedidentity" \
  "AzureWebJobsStorage__clientId=${IDENTITY_CLIENT_ID}" \
  "FUNCTIONS_WORKER_RUNTIME=dotnet-isolated" \
  "TASKHUB_NAME=my-taskhub"
```

### 3. Deploy the function app

```bash
func azure functionapp publish my-func-app
```

### 4. Test the deployment

```bash
curl -X POST https://my-func-app.azurewebsites.net/api/StartLargePayload
```

Poll the `statusQueryGetUri` from the response to check completion. A successful result looks like:

```json
{
  "runtimeStatus": "Completed",
  "output": {
    "ItemsProcessed": 5,
    "TotalSizeKb": 500,
    "IndividualSizes": [100, 100, 100, 100, 100]
  }
}
```

### 5. Verify payload externalization

Check that payloads were externalized to blob storage:

```bash
az storage blob list \
  --account-name mystorageaccount \
  --container-name durabletask-payloads \
  --auth-mode login \
  --output table
```

You should see compressed blobs (typically ~450 bytes each for 100 KB payloads due to gzip compression of repetitive data).

## Configuration options

| App Setting | Description | Example |
|---|---|---|
| `PAYLOAD_SIZE_KB` | Size of each generated payload in KB | `100` |
| `ACTIVITY_COUNT` | Number of parallel activity invocations | `5` |

## Next steps

- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
- [Durable Functions overview](https://learn.microsoft.com/azure/azure-functions/durable/durable-functions-overview)
- [Configure identity-based authentication](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler)
