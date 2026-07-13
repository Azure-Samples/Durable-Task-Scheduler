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
urlFragment: large-payload-fan-out-fan-in-dotnet
languages:
- csharp
- bicep
- azdeveloper
---
-->

# Large Payload Support — .NET Isolated Durable Functions (Fan-out/Fan-in)

This sample shows how Durable Functions can safely process orchestration data that is **larger than 1 MB** when Durable Task Scheduler is configured with **large payload storage**.

If you want the smallest possible example, start with the sibling [LargePayload](../LargePayload) sample. Both samples use the same DTS + blob storage configuration and deployment story; this folder keeps the original fan-out/fan-in orchestration shape.

The flow is intentionally parallel:

1. An HTTP trigger starts an orchestration with a payload size larger than 1 MB.
2. The orchestrator fans out to multiple activities.
3. Each activity generates a payload larger than 1 MB and returns it to the orchestrator.
4. The orchestrator fans in and returns a small summary proving every activity payload survived the round-trip.

This is exactly why large payload support exists: without blob offload, payloads this size would be too large to flow through DTS messages directly.

## How large payload storage works

The sample enables these settings in `host.json`:

```json
"durableTask": {
  "storageProvider": {
    "type": "azureManaged",
    "connectionStringName": "DTS_CONNECTION_STRING",
    "payloadStorageEnabled": true,
    "payloadStorageThresholdBytes": 262144,
    "payloadStorageMaxPayloadBytes": 15728640
  },
  "hubName": "%TASKHUB_NAME%"
}
```

When a payload exceeds `payloadStorageThresholdBytes`, the Durable Functions extension:

1. compresses the payload with gzip
2. stores it in blob storage using `AzureWebJobsStorage`
3. replaces the in-band DTS message with a small blob reference
4. resolves that blob reference automatically before your function code reads the payload

The sample uses **three deterministic, low-compressibility 1.5 MiB activity payloads** by default and a **262,144-byte (256 KiB)** threshold so externalization happens before the payload approaches the DTS 1 MiB message boundary.

### Maximum payload size

`payloadStorageThresholdBytes` controls *when* a payload is externalized. A separate setting, `payloadStorageMaxPayloadBytes`, controls the **largest single payload** the runtime will externalize:

- If you don't set `payloadStorageMaxPayloadBytes`, the default maximum is **10 MB (10,485,760 bytes, shown as 10,240 KB in the error message)**.
- The cap applies to each **single** externalized payload (here, each activity's payload), not the combined total across all activities.
- A request whose payload exceeds the maximum **fails fast** — before the orchestration is scheduled — with an error like:

  ```text
  Payload size 10742 KB exceeds the configured maximum of 10240 KB. Reduce the payload size or increase the max payload size limit.
  ```

- This is a **configurable limit, not a hard product limit**. To allow larger payloads, raise `payloadStorageMaxPayloadBytes` in `host.json`. This sample sets it to `15728640` (15 MiB), so each payload up to 15 MiB is accepted.

#### `PAYLOAD_SIZE_BYTES` is not `payloadStorageMaxPayloadBytes`

These two settings are easy to confuse but do completely different things:

| Setting | Where it lives | What it controls |
|---|---|---|
| `PAYLOAD_SIZE_BYTES` | app setting / `local.settings.json` | How many bytes of test data **each activity generates** for its demo payload. |
| `payloadStorageMaxPayloadBytes` | `host.json` | The **maximum** size of any single payload the runtime will externalize. Larger payloads fail fast. |

If you raise `PAYLOAD_SIZE_BYTES` above `payloadStorageMaxPayloadBytes` (for example `PAYLOAD_SIZE_BYTES=11000000` against the default 10 MB cap), the run fails **by design** with the error above. Raise `payloadStorageMaxPayloadBytes` to at least the per-activity payload size you want to support.

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
       "PAYLOAD_SIZE_BYTES": "1572864",
       "ACTIVITY_COUNT": "3"
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

5. Poll the `StatusQueryGetUri` value from the response until the orchestration completes. Focus on the `runtimeStatus` and `output` fields. The important part looks like this:

   ```json
   {
     "runtimeStatus": "Completed",
     "output": {
       "ActivityCount": 3,
       "RequestedPayloadBytesPerActivity": 1572864,
       "IndividualPayloadBytes": [1572864, 1572864, 1572864],
       "TotalPayloadBytes": 4718592,
       "AllPayloadsExceededOneMiB": true,
       "AllPayloadsMatchRequestedSize": true
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

The extension stores payload blobs with gzip content encoding, so Azure shows the compressed on-disk size. Because this sample uses low-compressibility payload content, the stored blob sizes should stay reasonably close to the logical 1.5 MiB activity payloads instead of collapsing to tiny repetitive-text blobs.

## Local settings

| Setting | Description | Default |
|---|---|---|
| `DTS_CONNECTION_STRING` | DTS emulator or Azure connection string | `Endpoint=http://localhost:8080;Authentication=None` |
| `TASKHUB_NAME` | Task hub name | `default` |
| `AzureWebJobsStorage` | Storage for Functions host state and payload blobs | `UseDevelopmentStorage=true` locally |
| `PAYLOAD_SIZE_BYTES` | Size of the test payload each activity generates. Keep it at or below the configured `payloadStorageMaxPayloadBytes` (default 10 MB); larger values fail by design. | `1572864` |
| `ACTIVITY_COUNT` | Number of parallel activity invocations | `3` |

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

- `LargePayloadOrchestration.cs` — HTTP starter, fan-out/fan-in orchestrator, and payload-generating activity
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
