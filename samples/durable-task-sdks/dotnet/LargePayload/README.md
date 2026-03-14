# Large Payload Support — Durable Task SDK (.NET)

## Description

This sample shows how the Durable Task SDK externalizes payloads to Azure Blob Storage so an orchestration can safely process data that is **larger than 1 MB**.

The flow is intentionally simple:

1. The client starts an orchestration with a payload larger than 1 MB.
2. The worker echoes that payload through an activity.
3. The sample prints whether payload blobs were created during the run.

This is the pattern you need when your durable workflow would otherwise hit the Durable Task Scheduler message-size limit.

## Why this sample exists

Durable Task Scheduler messages have a size limit. The SDK-side blob payload extension solves that by:

- uploading large payloads to blob storage
- replacing the in-band message with a small blob reference
- resolving that reference automatically before your orchestrator or activity code reads it

The sample uses a **1.5 MiB** payload by default and an offload threshold of **900,000 bytes** so payloads are externalized before they approach the 1 MiB scheduler ceiling.

> The SDK extension requires `EXTERNALIZE_THRESHOLD_BYTES` to stay at or below `1,048,576` bytes.

## Prerequisites

1. [.NET 10 SDK](https://dotnet.microsoft.com/download/dotnet/10.0)
2. [Docker](https://www.docker.com/products/docker-desktop/)
3. [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli) for the Azure path

## Run locally with DTS emulator + Azurite

1. Start the local dependencies:

   ```bash
   docker compose up -d
   ```

   This starts:

   - DTS emulator on `http://localhost:8080`
   - DTS dashboard on `http://localhost:8082`
   - Azurite blob/queue/table endpoints on `10000-10002`

2. Run the sample:

   ```bash
   dotnet run --project LargePayload.csproj
   ```

3. Expected output includes:

   - payload size larger than 1 MiB
   - `Payload blobs added during run: 1` or higher
   - `Payload offload observed: True`
   - `Round-trip payload matched: True`

4. Verify the payload blobs exist:

   ```bash
   az storage blob list \
     --connection-string "UseDevelopmentStorage=true" \
     --container-name durabletask-payloads \
     --output table
   ```

   Because the payload is repetitive text and the extension uses gzip compression, the stored blobs will be much smaller than the original 1.5 MiB payload.

## Local configuration

The sample works out of the box locally, but you can override the defaults with environment variables:

| Variable | Description | Default |
|---|---|---|
| `DURABLE_TASK_SCHEDULER_CONNECTION_STRING` | DTS connection string | `Endpoint=http://localhost:8080;TaskHub=default;Authentication=None` |
| `PAYLOAD_STORAGE_CONNECTION_STRING` | Storage connection string for payload blobs | `UseDevelopmentStorage=true` |
| `PAYLOAD_STORAGE_ACCOUNT_URI` | Blob account URI for identity-based storage access | unset |
| `PAYLOAD_CONTAINER_NAME` | Blob container used for externalized payloads | `durabletask-payloads` |
| `PAYLOAD_SIZE_BYTES` | Payload size for the orchestration input | `1572864` |
| `EXTERNALIZE_THRESHOLD_BYTES` | Blob offload threshold | `900000` |
| `PAYLOAD_STORAGE_MANAGED_IDENTITY_CLIENT_ID` | Optional user-assigned managed identity client ID for storage | unset |

If `PAYLOAD_STORAGE_CONNECTION_STRING` is not set and `PAYLOAD_STORAGE_ACCOUNT_URI` is provided, the sample uses `DefaultAzureCredential`.

## Run in Azure

1. Create the Azure resources:

   ```bash
   RESOURCE_GROUP=my-large-payload-rg
   LOCATION=eastus
   SCHEDULER_NAME=my-large-payload-dts
   TASKHUB_NAME=largepayload
   STORAGE_ACCOUNT=mylargepayloadsa

   az group create --name $RESOURCE_GROUP --location $LOCATION

   az storage account create \
     --name $STORAGE_ACCOUNT \
     --resource-group $RESOURCE_GROUP \
     --location $LOCATION \
     --sku Standard_LRS

   az durabletask scheduler create \
     --resource-group $RESOURCE_GROUP \
     --name $SCHEDULER_NAME \
     --location $LOCATION \
     --sku-name Consumption

   az durabletask taskhub create \
     --resource-group $RESOURCE_GROUP \
     --scheduler-name $SCHEDULER_NAME \
     --name $TASKHUB_NAME
   ```

2. Grant your signed-in identity access to DTS and blob storage:

   ```bash
   PRINCIPAL=$(az account show --query user.name -o tsv)
   SUBSCRIPTION_ID=$(az account show --query id -o tsv)
   STORAGE_ID=$(az storage account show --name $STORAGE_ACCOUNT --resource-group $RESOURCE_GROUP --query id -o tsv)

   az role assignment create \
     --assignee $PRINCIPAL \
     --role "Durable Task Data Contributor" \
     --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.DurableTask/schedulers/$SCHEDULER_NAME/taskHubs/$TASKHUB_NAME"

   az role assignment create \
     --assignee $PRINCIPAL \
     --role "Storage Blob Data Contributor" \
     --scope $STORAGE_ID
   ```

3. Set the environment variables and run the sample:

   ```bash
   SCHEDULER_ENDPOINT=$(az durabletask scheduler show \
     --resource-group $RESOURCE_GROUP \
     --name $SCHEDULER_NAME \
     --query properties.endpoint \
     --output tsv)

   export DURABLE_TASK_SCHEDULER_CONNECTION_STRING="Endpoint=$SCHEDULER_ENDPOINT;TaskHub=$TASKHUB_NAME;Authentication=DefaultAzure"
   export PAYLOAD_STORAGE_ACCOUNT_URI="https://$STORAGE_ACCOUNT.blob.core.windows.net"

   dotnet run --project LargePayload.csproj
   ```

4. Verify that externalized payloads were written to blob storage:

   ```bash
   az storage blob list \
     --account-name $STORAGE_ACCOUNT \
     --container-name durabletask-payloads \
     --auth-mode login \
     --output table
   ```

If you use a user-assigned managed identity, set `PAYLOAD_STORAGE_MANAGED_IDENTITY_CLIENT_ID` (or `AZURE_CLIENT_ID`) before running the sample.

## Clean up

Stop the local dependencies:

```bash
docker compose down
```

Delete Azure resources when you no longer need them:

```bash
az group delete --name $RESOURCE_GROUP --yes --no-wait
```
