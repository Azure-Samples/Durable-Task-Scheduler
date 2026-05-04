<!--
---
description: This sample demonstrates how to run Durable Functions with payloads larger than 1 MB by externalizing orchestration data to blob storage.
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
- bicep
- azdeveloper
---
-->

# Large Payload Support — .NET Isolated Durable Functions (Round-Trip)

This sample shows how Durable Functions can safely process orchestration data that is **larger than 1 MB** when Durable Task Scheduler is configured with **large payload storage**.

If you want the same storage feature demonstrated with a parallel fan-out/fan-in orchestration, see the sibling [LargePayloadFanOutFanIn](../LargePayloadFanOutFanIn) sample. Both samples use the same DTS + blob storage configuration and deployment story; this folder is the simplest place to start.

The flow is intentionally simple:

1. An HTTP trigger starts an orchestration with a payload larger than 1 MB.
2. The orchestrator sends that payload to a single activity.
3. The activity echoes the payload back.
4. The orchestration returns a small summary proving the payload survived the round-trip.

This is exactly why large payload support exists: without blob offload, a payload this size would be too large to flow through DTS messages directly.

## How large payload storage works

The sample enables these settings in `host.json`:

```json
"durableTask": {
  "storageProvider": {
    "type": "azureManaged",
    "connectionStringName": "DTS_CONNECTION_STRING",
    "payloadStorageEnabled": true,
    "payloadStorageThresholdBytes": 900000
  },
  "hubName": "%TASKHUB_NAME%"
}
```

When a payload exceeds `payloadStorageThresholdBytes`, the Durable Functions extension:

1. compresses the payload with gzip
2. stores it in blob storage using `AzureWebJobsStorage`
3. replaces the in-band DTS message with a small blob reference
4. resolves that blob reference automatically before your function code reads the payload

The sample uses a deterministic, low-compressibility **1.5 MiB** payload by default and a **900,000-byte** threshold so externalization happens before the payload approaches the DTS 1 MiB message boundary.

## Prerequisites

- [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0)
- [Azure Functions Core Tools v4](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
- [Docker](https://www.docker.com/products/docker-desktop/)
- [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli)
- [Azure Developer CLI (`azd`)](https://aka.ms/azd) for the Azure deployment path

## Run locally with DTS emulator + Azurite

1. Start the local dependencies:

   ```bash
   docker compose up -d
   ```

2. Create `local.settings.json`:

   ```json
   {
     "IsEncrypted": false,
     "Values": {
       "FUNCTIONS_WORKER_RUNTIME": "dotnet-isolated",
       "AzureWebJobsStorage": "UseDevelopmentStorage=true",
       "DTS_CONNECTION_STRING": "Endpoint=http://localhost:8080;Authentication=None",
       "TASKHUB_NAME": "default",
       "PAYLOAD_SIZE_BYTES": "1572864"
     }
   }
   ```

3. Start the function app:

   ```bash
   func start
   ```

4. Trigger the orchestration:

   ```bash
   curl -X POST http://localhost:7071/api/StartLargePayload
   ```

5. Poll the `StatusQueryGetUri` value from the response until the orchestration completes. The full status payload also includes the original large `input`, so focus on the `runtimeStatus` and `output` fields. The important part looks like this:

   ```json
   {
     "runtimeStatus": "Completed",
     "output": {
       "RequestedPayloadBytes": 1572864,
       "OrchestrationInputBytes": 1572864,
       "ActivityOutputBytes": 1572864,
       "ExceededOneMiB": true,
       "PayloadsMatch": true
     }
   }
   ```

6. Verify that blob offload happened:

   ```bash
   az storage blob list \
     --connection-string "UseDevelopmentStorage=true" \
     --container-name durabletask-payloads \
     --output table
   ```

The extension stores payload blobs with gzip content encoding, so Azure shows the compressed on-disk size. Because this sample uses low-compressibility payload content, the stored blob sizes should stay reasonably close to the logical 1.5 MiB payload instead of collapsing to a tiny repetitive-text blob.

## Local settings

| Setting | Description | Default |
|---|---|---|
| `DTS_CONNECTION_STRING` | DTS emulator or Azure connection string | `Endpoint=http://localhost:8080;Authentication=None` |
| `TASKHUB_NAME` | Task hub name | `default` |
| `AzureWebJobsStorage` | Storage for Functions host state and payload blobs | `UseDevelopmentStorage=true` locally |
| `PAYLOAD_SIZE_BYTES` | Payload size used by the HTTP starter | `1572864` |

## Deploy to Azure with AZD

This sample includes `azure.yaml`, `infra/`, and deployment scripts so you can provision the DTS + Function App + Storage resources together.

1. Sign in:

   ```bash
   az login
   azd auth login
   ```

2. From this sample directory, provision and deploy:

   ```bash
   azd up
   ```

   The deployment provisions:

   - an Azure Function App
   - a storage account used for `AzureWebJobsStorage`
   - a Durable Task Scheduler resource and task hub
   - a user-assigned managed identity with the required DTS and storage permissions

3. Load the environment values:

   ```bash
   eval "$(azd env get-values)"
   ```

4. Start the orchestration in Azure:

   ```bash
   curl -X POST "https://${AZURE_FUNCTION_NAME}.azurewebsites.net/api/StartLargePayload"
   ```

5. Verify payload blobs in Azure storage:

   ```bash
   az storage blob list \
     --account-name $AZURE_STORAGE_ACCOUNT_NAME \
     --container-name durabletask-payloads \
     --auth-mode login \
     --output table
   ```

6. Open the DTS dashboard URL in the Azure portal to inspect the orchestration history.

## Files to look at

- `LargePayloadOrchestration.cs` — HTTP starter, orchestrator, and echo activity
- `host.json` — Durable Task Scheduler + large payload configuration
- `docker-compose.yml` — local DTS emulator + Azurite dependencies
- `azure.yaml` and `infra/` — Azure deployment path

## Clean up

Stop the local containers:

```bash
docker compose down
```

Delete the Azure environment when you are done:

```bash
azd down --purge
```
