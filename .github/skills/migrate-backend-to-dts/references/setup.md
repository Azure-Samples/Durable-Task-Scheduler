# Durable Task Scheduler Setup Reference

Complete guide for provisioning and configuring the Durable Task Scheduler for migrated Durable Functions apps.

## Local Development with Emulator

### Prerequisites

```bash
# Docker (required for DTS emulator)
# Install from https://docs.docker.com/get-docker/

# Azure Functions Core Tools
brew tap azure/functions
brew install azure-functions-core-tools@4

# Azurite (Azure Storage emulator — still needed for Functions runtime)
npm install -g azurite
```

### Start the Emulator

```bash
# Pull and run the DTS emulator
docker run -d -p 8080:8080 -p 8082:8082 --name dts-emulator mcr.microsoft.com/dts/dts-emulator:latest

# Verify it's running
curl -s http://localhost:8082/health
```

- **Port 8080** — gRPC/HTTP endpoint (your app connects here)
- **Port 8082** — Dashboard UI (monitor orchestrations)
- **Dashboard:** http://localhost:8082

### Docker Compose (Emulator + Azurite)

```yaml
# docker-compose.yml
version: '3.8'

services:
  azurite:
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - "10000:10000"  # Blob
      - "10001:10001"  # Queue
      - "10002:10002"  # Table
    volumes:
      - azurite-data:/data
    command: azurite --blobHost 0.0.0.0 --queueHost 0.0.0.0 --tableHost 0.0.0.0

  dts-emulator:
    image: mcr.microsoft.com/dts/dts-emulator:latest
    ports:
      - "8080:8080"    # gRPC/HTTP endpoint
      - "8082:8082"    # Dashboard
    environment:
      - DTS_EMULATOR_LOG_LEVEL=Information
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/health"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  azurite-data:
```

```bash
docker-compose up -d
```

### Local Connection Strings

**local.settings.json:**

```json
{
  "IsEncrypted": false,
  "Values": {
    "AzureWebJobsStorage": "UseDevelopmentStorage=true",
    "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=http://localhost:8080;Authentication=None",
    "TASKHUB_NAME": "default"
  }
}
```

## Azure Provisioning

### Option 1: Azure Portal

1. Go to the [Azure portal](https://portal.azure.com)
2. Search for "Durable Task Scheduler" in the marketplace
3. Create a new scheduler resource:
   - Choose a **SKU** (Dedicated or Consumption)
   - Select a **region**
   - Provide a **name**
4. After creation, navigate to the resource and create a **Task Hub**
5. Copy the endpoint URL from the overview page

### Option 2: Azure CLI

```bash
# Install the Durable Task Scheduler extension (if not already installed)
az extension add --name durabletask

# Create a scheduler
az durabletask scheduler create \
  --resource-group <resource-group> \
  --name <scheduler-name> \
  --location <region> \
  --sku dedicated

# Create a task hub
az durabletask taskhub create \
  --resource-group <resource-group> \
  --scheduler-name <scheduler-name> \
  --name <taskhub-name>

# Get the endpoint
az durabletask scheduler show \
  --resource-group <resource-group> \
  --name <scheduler-name> \
  --query endpoint -o tsv
```

### Option 3: Bicep

```bicep
resource scheduler 'Microsoft.DurableTask/schedulers@2025-04-01-preview' = {
  name: schedulerName
  location: location
  properties: {
    sku: {
      name: 'Dedicated'
      capacity: 1
    }
  }
}

resource taskHub 'Microsoft.DurableTask/schedulers/taskHubs@2025-04-01-preview' = {
  parent: scheduler
  name: taskHubName
}
```

## Identity & Authentication

DTS uses identity-based authentication exclusively. No shared keys or secret-bearing connection strings.

### Configure Managed Identity

```bash
# Enable system-assigned managed identity on your Function App
az functionapp identity assign \
  --resource-group <resource-group> \
  --name <function-app-name>

# Get the principal ID
PRINCIPAL_ID=$(az functionapp identity show \
  --resource-group <resource-group> \
  --name <function-app-name> \
  --query principalId -o tsv)
```

### Assign RBAC Role

```bash
# Get the DTS scheduler resource ID
SCHEDULER_ID=$(az durabletask scheduler show \
  --resource-group <resource-group> \
  --name <scheduler-name> \
  --query id -o tsv)

# Assign the Task Hub Contributor role
az role assignment create \
  --assignee "$PRINCIPAL_ID" \
  --role "Durable Task Scheduler Task Hub Contributor" \
  --scope "$SCHEDULER_ID"
```

### Connection String Format

| Environment | Connection String |
|-------------|-------------------|
| **Local emulator** | `Endpoint=http://localhost:8080;Authentication=None` |
| **Azure (managed identity)** | `Endpoint=https://<scheduler>.<region>.durabletask.io;Authentication=DefaultAzure` |
| **Azure (specific task hub)** | `Endpoint=https://<scheduler>.<region>.durabletask.io;TaskHub=<name>;Authentication=DefaultAzure` |

### Set App Settings

```bash
az functionapp config appsettings set \
  --resource-group <resource-group> \
  --name <function-app-name> \
  --settings \
    "DURABLE_TASK_SCHEDULER_CONNECTION_STRING=Endpoint=https://<scheduler>.<region>.durabletask.io;Authentication=DefaultAzure" \
    "TASKHUB_NAME=<taskhub-name>"
```

## Large Payload Storage

DTS has a message size limit. For orchestrations that pass large inputs/outputs, enable blob-based overflow:

### Configuration

```json
{
  "extensions": {
    "durableTask": {
      "storageProvider": {
        "type": "azureManaged",
        "connectionStringName": "DURABLE_TASK_SCHEDULER_CONNECTION_STRING",
        "largePayloadStorageEnabled": true,
        "largePayloadStorageThresholdBytes": 10240
      },
      "hubName": "%TASKHUB_NAME%"
    }
  }
}
```

- **`largePayloadStorageEnabled`** — set to `true` to enable blob offload
- **`largePayloadStorageThresholdBytes`** — payloads larger than this (default 10240 = 10 KB) are stored in blob

Large payloads are stored in the Azure Storage account referenced by `AzureWebJobsStorage`.

## Monitoring

### DTS Dashboard

- **Azure:** [dashboard.durabletask.io](https://dashboard.durabletask.io) — view orchestration status, history, and perform management operations
- **Local emulator:** http://localhost:8082

### Application Insights

Durable Functions continues to emit telemetry to Application Insights when configured. No changes needed for monitoring integration.

### Distributed Tracing

DTS supports distributed tracing v2 with Application Insights. Enable in host.json:

```json
{
  "extensions": {
    "durableTask": {
      "tracing": {
        "distributedTracingEnabled": true,
        "distributedTracingProtocol": "W3CTraceContext"
      }
    }
  }
}
```

## Troubleshooting

### Emulator Won't Start

```bash
# Check if ports are in use
lsof -i :8080
lsof -i :8082

# Pull latest image
docker pull mcr.microsoft.com/dts/dts-emulator:latest

# Remap ports if needed
docker run -d -p 9080:8080 -p 9082:8082 --name dts-emulator mcr.microsoft.com/dts/dts-emulator:latest
# Update connection string: Endpoint=http://localhost:9080;Authentication=None
```

### Authentication Errors in Azure

```bash
# Verify managed identity is enabled
az functionapp identity show --resource-group <rg> --name <app>

# Verify role assignment exists
az role assignment list --assignee <principal-id> --scope <scheduler-resource-id>

# Check the role name is correct
# Must be: "Durable Task Scheduler Task Hub Contributor"
```

### Orchestrations Not Starting

1. Verify `host.json` has correct `storageProvider.type` (`azureManaged`)
2. Verify connection string is accessible (check app settings)
3. Check Function App logs for connection errors
4. Verify task hub exists in the scheduler resource
5. For non-.NET: verify extension bundle is `Microsoft.Azure.Functions.ExtensionBundle.Preview` with version `[4.29.0, 5.0.0)`
