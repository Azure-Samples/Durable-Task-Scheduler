# Function Chaining Pattern

## Description of the Sample

This sample demonstrates the function chaining pattern with the Azure Durable Task Scheduler using the JavaScript SDK. Function chaining is a fundamental workflow pattern where activities are executed in a sequence, with the output of one activity passed as the input to the next activity.

In this sample:

1. The orchestrator calls the `sayHello` activity with a name input.
2. The result is passed to the `processGreeting` activity.
3. That result is passed to the `finalizeResponse` activity.
4. The final result is returned to the client.

This pattern is useful for:

- Creating sequential workflows where steps must execute in order
- Passing data between steps with data transformations at each step
- Building pipelines where each activity adds value to the result

## Prerequisites

1. [Node.js 22+](https://nodejs.org/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator) installed
3. [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)
4. [Azure Developer CLI](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/install-azd)

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

Note: The sample code automatically uses the default emulator settings (endpoint: `http://localhost:8080`, taskhub: `default`). You don't need to set any environment variables.

### Using a Deployed Scheduler and Taskhub in Azure

1. Install the durable task scheduler CLI extension:

   ```bash
   az upgrade
   az extension add --name durabletask --allow-preview true
   ```

2. Create a resource group in a region where the Durable Task Scheduler is available:

   ```bash
   az provider show --namespace Microsoft.DurableTask --query "resourceTypes[?resourceType=='schedulers'].locations | [0]" --out table
   az group create --name my-resource-group --location <location>
   ```

3. Create a durable task scheduler resource:

   ```bash
   az durabletask scheduler create \
     --resource-group my-resource-group \
     --name my-scheduler \
     --ip-allowlist '["0.0.0.0/0"]' \
     --sku-name Dedicated \
     --sku-capacity 1
   ```

4. Create a task hub within the scheduler resource:

   ```bash
   az durabletask taskhub create \
     --resource-group my-resource-group \
     --scheduler-name my-scheduler \
     --name my-taskhub
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

6. Set environment variables in each terminal where you run the worker or client:

   ```bash
   export ENDPOINT=$(az durabletask scheduler show \
     --resource-group my-resource-group \
     --name my-scheduler \
     --query "properties.endpoint" \
     --output tsv)

   export TASKHUB="my-taskhub"
   ```

## How to Run the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. Navigate to this sample directory:

   ```bash
   cd samples/durable-task-sdks/javascript/function-chaining
   ```

2. Install dependencies:

   ```bash
   npm install
   ```

3. Start the worker in a terminal:

   ```bash
   npm run worker
   ```

4. In a new terminal, run the client:

   ```bash
   npm run client
   ```

   You can optionally provide a base name:

   ```bash
   npm run client -- Alice
   ```

The client schedules 20 orchestrations, one every 5 seconds, then waits for all of them to complete.

## Deploying with Azure Developer CLI (AZD)

This sample includes an `azure.yaml` configuration file that allows you to deploy the entire solution to Azure using Azure Developer CLI (AZD).

> **Note:** This sample uses the shared infrastructure templates located at [`samples/infra/`](../../../infra/).

### Deployment Steps

1. Authenticate with Azure:

   ```bash
   azd auth login
   ```

2. Navigate to the Function Chaining sample directory:

   ```bash
   cd samples/durable-task-sdks/javascript/function-chaining
   ```

3. Initialize the Azure Developer CLI project (only needed the first time):

   ```bash
   azd init
   ```

4. Provision resources and deploy the application:

   ```bash
   azd up
   ```

This command provisions Azure resources (including Azure Container Apps and Durable Task Scheduler), builds and deploys both the client and worker, and sets up the required environment configuration.

## Confirm Successful Deployment

1. In the Azure portal, open the resource group created by `azd up`.
2. Open the `client` container app and select **Monitoring > Log stream**.
3. Confirm orchestrations are being scheduled and completed.
4. Open the `worker` container app and select **Monitoring > Log stream**.
5. Confirm activities are being executed in order.

## Review Orchestration Status and History

To inspect orchestration history:

- Local emulator: open `http://localhost:8082` and select the `default` task hub.
- Deployed scheduler: open the task hub dashboard URL from your scheduler resource in Azure.

The dashboard shows the sequence of activities and input/output transitions for each orchestration instance.
