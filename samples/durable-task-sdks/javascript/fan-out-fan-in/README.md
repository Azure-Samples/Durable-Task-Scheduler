# Fan-out/Fan-in Pattern

## Description of the Sample

This sample demonstrates the fan-out/fan-in pattern with Azure Durable Task Scheduler using the JavaScript SDK.

In this sample:

1. The orchestrator receives a list of work items as input.
2. It fans out by scheduling `processWorkItem` activities in parallel.
3. It waits for all activity tasks to complete using `whenAll`.
4. It fans in by calling `aggregateResults` to compute totals.
5. The aggregated result is returned to the client.

This pattern is useful for:

- Parallel batch processing
- Workload distribution across independent tasks
- Aggregating results from concurrent activities

## Prerequisites

1. [Node.js 22+](https://nodejs.org/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli) (optional, if using a deployed Durable Task Scheduler)

## Set up the Durable Task Scheduler Emulator

The sample checks for deployed scheduler settings in environment variables. If none are present, it uses the local emulator.

1. From the repository root, navigate to this sample:

   ```bash
   cd samples/durable-task-sdks/javascript/fan-out-fan-in
   ```

2. Pull the emulator image:

   ```bash
   docker pull mcr.microsoft.com/dts/dts-emulator:latest
   ```

3. Run the emulator:

   ```bash
   docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
   ```

Default local settings used by this sample:

- Endpoint: `http://localhost:8080`
- Task hub: `default`

## Using a Deployed Scheduler and Task Hub in Azure

For local development against deployed Durable Task Scheduler resources:

1. Install the Durable Task CLI extension:

   ```bash
   az upgrade
   az extension add --name durabletask --allow-preview true
   ```

2. Create a resource group in a supported region:

   ```bash
   az provider show --namespace Microsoft.DurableTask --query "resourceTypes[?resourceType=='schedulers'].locations | [0]" --out table
   az group create --name my-resource-group --location <location>
   ```

3. Create a scheduler and task hub:

   ```bash
   az durabletask scheduler create \
     --resource-group my-resource-group \
     --name my-scheduler \
     --ip-allowlist '["0.0.0.0/0"]' \
     --sku-name Dedicated \
     --sku-capacity 1

   az durabletask taskhub create \
     --resource-group my-resource-group \
     --scheduler-name my-scheduler \
     --name my-taskhub
   ```

4. Assign yourself `Durable Task Data Contributor` on the task hub scope:

   ```bash
   subscriptionId=$(az account show --query "id" -o tsv)
   loggedInUser=$(az account show --query "user.name" -o tsv)

   az role assignment create \
     --assignee $loggedInUser \
     --role "Durable Task Data Contributor" \
     --scope "/subscriptions/$subscriptionId/resourceGroups/my-resource-group/providers/Microsoft.DurableTask/schedulers/my-scheduler/taskHubs/my-taskhub"
   ```

5. Set environment variables in each terminal where you run `worker` or `client`:

   ```bash
   export ENDPOINT=$(az durabletask scheduler show \
     --resource-group my-resource-group \
     --name my-scheduler \
     --query "properties.endpoint" \
     --output tsv)

   export TASKHUB="my-taskhub"
   ```

## Run the Quickstart

1. Install dependencies:

   ```bash
   npm install
   ```

2. Start the worker:

   ```bash
   npm run worker
   ```

3. In a separate terminal, run the client:

   ```bash
   npm run client
   ```

   Optionally, pass the number of work items:

   ```bash
   npm run client -- 15
   ```

## Understanding the Output

When you run the sample, output is generated from both processes.

### Worker output

The worker shows:

- Worker startup and scheduler connection details
- Parallel execution of `processWorkItem` activities
- Aggregation via `aggregateResults`

### Client output

The client shows:

- Scheduled orchestration instance ID
- Final orchestration runtime status
- Aggregated output JSON (total items, sum, average, and per-item results)

## Review orchestration status and history

You can inspect orchestration history in the dashboard:

1. Open `http://localhost:8082`
2. Select the `default` task hub
3. Open the orchestration instance to inspect activity fan-out and aggregation details

## Deploying with Azure Developer CLI (AZD)

This sample includes an `azure.yaml` file to deploy both worker and client to Azure Container Apps.

> **Note:** This sample uses the shared infrastructure templates at [`samples/infra/`](../../../infra/).

### AZD prerequisites

1. Install [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd)
2. Authenticate:

   ```bash
   azd auth login
   ```

### AZD deployment steps

1. From this sample directory, initialize once:

   ```bash
   azd init
   ```

2. Provision infrastructure and deploy:

   ```bash
   azd up
   ```

   This command provisions shared infrastructure (including Durable Task Scheduler and Azure Container Apps), builds container images, and deploys both services.

3. After deployment, monitor execution using:

   - Azure Portal Container App log streams for `worker` and `client`
   - Durable Task Scheduler dashboard URL from the deployed task hub
