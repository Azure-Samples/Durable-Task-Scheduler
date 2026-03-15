# Large Payload Support — Durable Task SDK (.NET)

## Description

This sample shows how the Durable Task SDK externalizes payloads to Azure Blob Storage so an orchestration can safely process data that is **larger than 1 MB**.

The flow is intentionally simple:

1. An HTTP endpoint starts an orchestration with a payload larger than 1 MB.
2. The worker echoes that payload through an activity.
3. The HTTP response reports whether payload blobs were created during the run, how many stored bytes were added, and whether the payload survived the round-trip.

This is the pattern you need when your durable workflow would otherwise hit the Durable Task Scheduler message-size limit.

## Why this sample exists

Durable Task Scheduler messages have a size limit. The SDK-side blob payload extension solves that by:

- uploading large payloads to blob storage
- replacing the in-band message with a small blob reference
- resolving that reference automatically before your orchestrator or activity code reads it

The sample uses a deterministic, low-compressibility **1.5 MiB** payload by default and an offload threshold of **900,000 bytes** so payloads are externalized before they approach the 1 MiB scheduler ceiling.

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
   ASPNETCORE_URLS=http://127.0.0.1:5098 dotnet run --project LargePayload.csproj
   ```

3. Trigger the round-trip verification:

   ```bash
   curl -X POST http://127.0.0.1:5098/api/largepayload/run \
     -H "Content-Type: application/json" \
     -d '{}'
   ```

    Expected response fields include:

    - `"runtimeStatus": "Completed"`
    - `"payloadExceedsOneMiB": true`
    - `"payloadOffloadObserved": true`
    - `"payloadStoredBytesAddedDuringRun"` is a large positive value
    - `"roundTripPayloadMatched": true`

4. Optional: query the status endpoint later with the returned `instanceId`:

   ```bash
   curl http://127.0.0.1:5098/api/largepayload/instances/<instanceId>
   ```

5. Verify the payload blobs exist:

   ```bash
   az storage blob list \
     --connection-string "UseDevelopmentStorage=true" \
     --container-name durabletask-payloads \
     --output table
   ```

    The extension stores payload blobs with gzip content encoding, so Azure shows the compressed on-disk size. Because this sample uses low-compressibility payload content, the stored blob sizes should stay reasonably close to the logical 1.5 MiB payload instead of collapsing to a tiny repetitive-text blob.

## Local configuration

The sample works out of the box locally, but you can override the defaults with environment variables:

| Variable | Description | Default |
|---|---|---|
| `DURABLE_TASK_SCHEDULER_CONNECTION_STRING` | DTS connection string | `Endpoint=http://localhost:8080;TaskHub=default;Authentication=None` |
| `PAYLOAD_STORAGE_CONNECTION_STRING` | Storage connection string for payload blobs | `UseDevelopmentStorage=true` |
| `PAYLOAD_STORAGE_ACCOUNT_URI` | Blob account URI for identity-based storage access | unset |
| `PAYLOAD_CONTAINER_NAME` | Blob container used for externalized payloads | `durabletask-payloads` |
| `PAYLOAD_SIZE_BYTES` | Default payload size used by the run endpoint | `1572864` |
| `EXTERNALIZE_THRESHOLD_BYTES` | Blob offload threshold | `900000` |
| `PAYLOAD_STORAGE_MANAGED_IDENTITY_CLIENT_ID` | Optional user-assigned managed identity client ID for storage | unset |
| `ASPNETCORE_URLS` | Listen URLs for the web host | framework default |

If `PAYLOAD_STORAGE_CONNECTION_STRING` is not set and `PAYLOAD_STORAGE_ACCOUNT_URI` is provided, the sample uses `DefaultAzureCredential`.

## Deploy to Azure Container Apps with an existing scheduler

This sample includes a `Dockerfile` so it can run as a long-lived HTTP host in Azure Container Apps while still using the same orchestration and payload-offload logic.

The flow below assumes you already have:

- an existing Durable Task Scheduler resource
- an existing Container Apps environment
- an existing user-assigned managed identity
- an existing Azure Container Registry
- a storage account for payload blobs

1. Set deployment variables:

   ```bash
   RESOURCE_GROUP=my-containerapps-rg
   CONTAINERAPPS_ENV=my-containerapps-env
   CONTAINER_APP_NAME=largepayload-sample
   REGISTRY_NAME=myregistry
   REGISTRY_LOGIN_SERVER=$(az acr show --name $REGISTRY_NAME --query loginServer -o tsv)
   IDENTITY_RESOURCE_GROUP=my-identity-rg
   IDENTITY_NAME=my-user-assigned-identity
   SCHEDULER_RESOURCE_GROUP=my-dts-rg
   SCHEDULER_NAME=my-large-payload-dts
   TASKHUB_NAME=largepayload
   STORAGE_RESOURCE_GROUP=my-storage-rg
   STORAGE_ACCOUNT=myexistingstorage
   IMAGE_TAG=largepayload:$(date +%Y%m%d%H%M%S)
   ```

2. Resolve the target resource IDs and create the task hub if needed:

   ```bash
   SUBSCRIPTION_ID=$(az account show --query id -o tsv)
   SCHEDULER_ENDPOINT=$(az durabletask scheduler show \
     --resource-group $SCHEDULER_RESOURCE_GROUP \
     --name $SCHEDULER_NAME \
     --query properties.endpoint \
     --output tsv)

   az durabletask taskhub show \
     --resource-group $SCHEDULER_RESOURCE_GROUP \
     --scheduler-name $SCHEDULER_NAME \
     --name $TASKHUB_NAME \
     --output none 2>/dev/null || \
   az durabletask taskhub create \
     --resource-group $SCHEDULER_RESOURCE_GROUP \
     --scheduler-name $SCHEDULER_NAME \
     --name $TASKHUB_NAME

   IDENTITY_ID=$(az identity show \
     --resource-group $IDENTITY_RESOURCE_GROUP \
     --name $IDENTITY_NAME \
     --query id \
     --output tsv)

   IDENTITY_CLIENT_ID=$(az identity show \
     --resource-group $IDENTITY_RESOURCE_GROUP \
     --name $IDENTITY_NAME \
     --query clientId \
     --output tsv)

   IDENTITY_PRINCIPAL_ID=$(az identity show \
     --resource-group $IDENTITY_RESOURCE_GROUP \
     --name $IDENTITY_NAME \
     --query principalId \
     --output tsv)

   STORAGE_ID=$(az storage account show \
     --name $STORAGE_ACCOUNT \
     --resource-group $STORAGE_RESOURCE_GROUP \
     --query id \
     --output tsv)
   ```

3. Grant the managed identity access to DTS and blob storage:

   ```bash
   TASKHUB_SCOPE="/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$SCHEDULER_RESOURCE_GROUP/providers/Microsoft.DurableTask/schedulers/$SCHEDULER_NAME/taskHubs/$TASKHUB_NAME"

   az role assignment create \
     --assignee-object-id $IDENTITY_PRINCIPAL_ID \
     --role "Durable Task Data Contributor" \
     --scope "$TASKHUB_SCOPE"

   az role assignment create \
     --assignee-object-id $IDENTITY_PRINCIPAL_ID \
     --role "Storage Blob Data Contributor" \
     --scope $STORAGE_ID
   ```

   If the assignments already exist, Azure CLI will report that and you can continue.

   If the storage account has a firewall (`defaultAction: Deny`), make sure the Container Apps environment can reach it. RBAC alone is not enough. In practice, that means either:

   - allowing the Container App outbound IPs on the storage account firewall, or
   - using a compatible VNet/subnet rule for the storage account and Container Apps environment

   Without that network access, the sample will fail with blob-storage `403 AuthorizationFailure` errors before the orchestration is scheduled.

4. Build the image in Azure Container Registry:

   ```bash
   az acr build \
     --registry $REGISTRY_NAME \
     --image $IMAGE_TAG \
     .
   ```

5. Create or update the Container App:

   ```bash
   FQDN=$(az containerapp create \
     --name $CONTAINER_APP_NAME \
     --resource-group $RESOURCE_GROUP \
     --environment $CONTAINERAPPS_ENV \
     --image $REGISTRY_LOGIN_SERVER/$IMAGE_TAG \
     --target-port 8080 \
     --ingress external \
     --min-replicas 1 \
     --max-replicas 1 \
     --user-assigned $IDENTITY_ID \
     --registry-server $REGISTRY_LOGIN_SERVER \
     --registry-identity $IDENTITY_ID \
     --env-vars \
       DURABLE_TASK_SCHEDULER_CONNECTION_STRING="Endpoint=$SCHEDULER_ENDPOINT;TaskHub=$TASKHUB_NAME;Authentication=ManagedIdentity;ClientID=$IDENTITY_CLIENT_ID" \
       AZURE_CLIENT_ID=$IDENTITY_CLIENT_ID \
       PAYLOAD_STORAGE_ACCOUNT_URI="https://$STORAGE_ACCOUNT.blob.core.windows.net" \
     --query properties.configuration.ingress.fqdn \
     --output tsv)

   echo "https://$FQDN"
   ```

   For a subsequent image refresh, use `az containerapp update --image $REGISTRY_LOGIN_SERVER/$IMAGE_TAG`.

6. Trigger the verification run:

   ```bash
   curl -X POST "https://$FQDN/api/largepayload/run" \
     -H "Content-Type: application/json" \
     -d '{}'
   ```

7. Verify that externalized payloads were written to blob storage:

   ```bash
   az storage blob list \
     --account-name $STORAGE_ACCOUNT \
     --container-name durabletask-payloads \
     --auth-mode login \
     --output table
   ```

8. Open the DTS dashboard URL for the task hub to inspect the orchestration history:

   ```bash
   az durabletask taskhub show \
     --resource-group $SCHEDULER_RESOURCE_GROUP \
     --scheduler-name $SCHEDULER_NAME \
     --name $TASKHUB_NAME \
     --query properties.dashboardUrl \
     --output tsv
   ```

## Clean up

Stop the local dependencies:

```bash
docker compose down
```

Delete the Container App when you no longer need it:

```bash
az containerapp delete --name $CONTAINER_APP_NAME --resource-group $RESOURCE_GROUP --yes
```
