# Durable Task Scheduler Setup and Deployment

## Local Development with Emulator

### Docker Setup

```bash
# Pull the emulator image
docker pull mcr.microsoft.com/dts/dts-emulator:latest

# Run the emulator
docker run -d \
  -p 8080:8080 \
  -p 8082:8082 \
  --name dts-emulator \
  mcr.microsoft.com/dts/dts-emulator:latest

# Emulator endpoints:
# - gRPC: http://localhost:8080
# - Dashboard: http://localhost:8082
```

### Docker Compose

```yaml
version: '3.8'
services:
  dts-emulator:
    image: mcr.microsoft.com/dts/dts-emulator:latest
    ports:
      - "8080:8080"  # gRPC endpoint
      - "8082:8082"  # Dashboard
    restart: unless-stopped
```

### Default Emulator Configuration

```javascript
const connectionString = "Endpoint=http://localhost:8080;Authentication=None;TaskHub=default";

const client = createAzureManagedClient(connectionString);
const workerBuilder = createAzureManagedWorkerBuilder(connectionString);
```

## Azure Durable Task Scheduler Provisioning

### Prerequisites

```bash
# Install Azure CLI
# https://learn.microsoft.com/cli/azure/install-azure-cli

# Login to Azure
az login

# Install durabletask extension
az extension add --name durabletask
```

### Create Scheduler and Task Hub

```bash
# Set variables
RESOURCE_GROUP="my-rg"
SCHEDULER_NAME="my-scheduler"
TASKHUB_NAME="my-taskhub"
LOCATION="eastus"

# Create resource group
az group create --name $RESOURCE_GROUP --location $LOCATION

# Create scheduler
az durabletask scheduler create \
  --resource-group $RESOURCE_GROUP \
  --name $SCHEDULER_NAME \
  --location $LOCATION \
  --ip-allowlist "[0.0.0.0/0]" \
  --sku-capacity 1 \
  --sku-name "Dedicated" \
  --tags "environment=dev"

# Create task hub
az durabletask taskhub create \
  --resource-group $RESOURCE_GROUP \
  --scheduler-name $SCHEDULER_NAME \
  --name $TASKHUB_NAME

# Get endpoint URL
az durabletask scheduler show \
  --resource-group $RESOURCE_GROUP \
  --name $SCHEDULER_NAME \
  --query "properties.endpoint" -o tsv
```

### Assign Permissions

```bash
# Get scheduler resource ID
SCHEDULER_ID=$(az durabletask scheduler show \
  --resource-group $RESOURCE_GROUP \
  --name $SCHEDULER_NAME \
  --query id -o tsv)

# Assign Durable Task Data Contributor role
az role assignment create \
  --assignee $(az ad signed-in-user show --query id -o tsv) \
  --role "Durable Task Data Contributor" \
  --scope $SCHEDULER_ID
```

## Application Configuration

### Environment Variables

```bash
# Connection string approach (recommended)
export DURABLE_TASK_SCHEDULER_CONNECTION_STRING="Endpoint=https://my-scheduler.region.durabletask.io;Authentication=DefaultAzure;TaskHub=my-hub"

# Or explicit parameters
export ENDPOINT="https://my-scheduler.region.durabletask.io"
export TASKHUB="my-taskhub"
```

### Configuration Helper

```javascript
import { DefaultAzureCredential, ManagedIdentityCredential } from "@azure/identity";
import { createAzureManagedClient, createAzureManagedWorkerBuilder } from "@microsoft/durabletask-js-azuremanaged";

const EMULATOR_ENDPOINT = "http://localhost:8080";

function getConnectionConfig() {
  const endpoint = process.env.ENDPOINT ?? EMULATOR_ENDPOINT;
  const taskHub = process.env.TASKHUB ?? "default";
  const managedIdentityClientId = process.env.AZURE_MANAGED_IDENTITY_CLIENT_ID;

  if (endpoint === EMULATOR_ENDPOINT) {
    return {
      connectionString: `Endpoint=${endpoint};Authentication=None;TaskHub=${taskHub}`,
    };
  }

  const credential = managedIdentityClientId
    ? new ManagedIdentityCredential({ clientId: managedIdentityClientId })
    : new DefaultAzureCredential();

  return { endpoint, taskHub, credential };
}

function createClientFromConfig() {
  const config = getConnectionConfig();
  if (config.connectionString) {
    return createAzureManagedClient(config.connectionString);
  }
  return createAzureManagedClient(config.endpoint, config.taskHub, config.credential);
}

function createWorkerBuilderFromConfig() {
  const config = getConnectionConfig();
  if (config.connectionString) {
    return createAzureManagedWorkerBuilder(config.connectionString);
  }
  return createAzureManagedWorkerBuilder(config.endpoint, config.taskHub, config.credential);
}
```

## Authentication Options

### Local Development (No Auth)

```javascript
const connectionString = "Endpoint=http://localhost:8080;Authentication=None;TaskHub=default";
```

### DefaultAzureCredential (Recommended for Azure)

```javascript
import { DefaultAzureCredential } from "@azure/identity";

const credential = new DefaultAzureCredential();
const client = createAzureManagedClient(endpoint, taskHub, credential);
```

### Managed Identity

```javascript
import { ManagedIdentityCredential } from "@azure/identity";

// System-assigned managed identity
const credential = new ManagedIdentityCredential();

// User-assigned managed identity
const credential = new ManagedIdentityCredential({ clientId: "<client-id>" });
```

### Connection String Authentication Types

Supported `Authentication` values in connection strings:
- `None` - No authentication (local emulator)
- `DefaultAzure` - DefaultAzureCredential
- `ManagedIdentity` - Managed Identity
- `WorkloadIdentity` - Workload Identity
- `Environment` - Environment variables
- `AzureCli` - Azure CLI
- `AzurePowerShell` - Azure PowerShell
- `VisualStudioCode` - VS Code
- `InteractiveBrowser` - Browser login

## Project Templates

### Worker Application

```javascript
// worker.mjs
import { createAzureManagedWorkerBuilder } from "@microsoft/durabletask-js-azuremanaged";

const connectionString = process.env.DURABLE_TASK_SCHEDULER_CONNECTION_STRING
  ?? "Endpoint=http://localhost:8080;Authentication=None;TaskHub=default";

// Define activities
const myActivity = async (_ctx, input) => {
  return `Processed: ${input}`;
};

// Define orchestrator
const myOrchestrator = async function* myOrchestrator(ctx, input) {
  const result = yield ctx.callActivity(myActivity, input);
  return result;
};

// Build and start worker
let worker;

async function stopWorker(exitCode = 0) {
  if (worker) {
    console.log("Stopping worker...");
    await worker.stop();
  }
  process.exit(exitCode);
}

process.on("SIGINT", async () => await stopWorker(0));
process.on("SIGTERM", async () => await stopWorker(0));

(async () => {
  worker = createAzureManagedWorkerBuilder(connectionString)
    .addOrchestrator(myOrchestrator)
    .addActivity(myActivity)
    .build();

  try {
    await worker.start();
    console.log("Worker started and waiting for orchestrations...");
    setInterval(() => {}, 60_000); // Keep process running
  } catch (error) {
    console.error("Worker failed to start", error);
    await stopWorker(1);
  }
})();
```

### Client Application

```javascript
// client.mjs
import { OrchestrationStatus } from "@microsoft/durabletask-js";
import { createAzureManagedClient } from "@microsoft/durabletask-js-azuremanaged";

const connectionString = process.env.DURABLE_TASK_SCHEDULER_CONNECTION_STRING
  ?? "Endpoint=http://localhost:8080;Authentication=None;TaskHub=default";

(async () => {
  const client = createAzureManagedClient(connectionString);

  try {
    const instanceId = await client.scheduleNewOrchestration("myOrchestrator", "Hello World");
    console.log(`Started orchestration: ${instanceId}`);

    const state = await client.waitForOrchestrationCompletion(instanceId, true, 120);

    if (state && state.runtimeStatus === OrchestrationStatus.COMPLETED) {
      console.log(`Result: ${state.serializedOutput}`);
    } else {
      console.log(`Orchestration finished with status: ${state?.runtimeStatus}`);
    }
  } finally {
    await client.stop();
  }
})();
```

## Deployment Options

### Docker Image

```dockerfile
FROM node:22-slim

WORKDIR /app

COPY package*.json ./
RUN npm ci --production

COPY . .

CMD ["node", "worker.mjs"]
```

### Azure Container Apps

```yaml
properties:
  configuration:
    secrets:
      - name: dts-connection-string
        value: "Endpoint=https://my-scheduler.region.durabletask.io;Authentication=ManagedIdentity;TaskHub=my-hub"
  template:
    containers:
      - image: myregistry.azurecr.io/durable-worker:latest
        name: worker
        env:
          - name: DURABLE_TASK_SCHEDULER_CONNECTION_STRING
            secretRef: dts-connection-string
        resources:
          cpu: 0.5
          memory: 1Gi
    scale:
      minReplicas: 1
      maxReplicas: 10
```

## Monitoring

### Dashboard Access

- **Emulator**: http://localhost:8082
- **Azure**: Navigate to Scheduler → Task Hub → Dashboard URL in portal

### Query Orchestration Status

```javascript
// Check specific instance
const state = await client.getOrchestrationState(instanceId, true);
console.log(`Status: ${state.runtimeStatus}`);
console.log(`Created: ${state.createdAt}`);
console.log(`Updated: ${state.lastUpdatedAt}`);
console.log(`Input: ${state.serializedInput}`);
console.log(`Output: ${state.serializedOutput}`);
```
