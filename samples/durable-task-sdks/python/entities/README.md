# Durable Entities Pattern

## Description of the Sample

This sample demonstrates the Durable Entities pattern with the Azure Durable Task Scheduler using the Python SDK. Durable entities are stateful objects that maintain state across operations and can be accessed by orchestrations or directly by clients.

In this sample:
1. A counter entity is defined that supports `add`, `subtract`, `get`, and `reset` operations
2. The client signals the entity directly to modify its state
3. Orchestrations interact with entities using `signal_entity` and `call_entity`
4. Entity state is automatically persisted and survives restarts

This pattern is useful for:
- Building aggregators and accumulators
- Maintaining shared state across workflows
- Implementing distributed counters, caches, or locks
- Creating actor-like programming models

## Prerequisites

1. [Python 3.9+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator) installed
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample locally:

### Using the Emulator (Recommended)

The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. Pull the Docker Image for the Emulator:
  ```bash
  docker pull mcr.microsoft.com/dts/dts-emulator:latest
  ```

1. Run the Emulator:
  ```bash
  docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
  ```
Wait a few seconds for the container to be ready.

Note: The example code automatically uses the default emulator settings (endpoint: http://localhost:8080, taskhub: default). You don't need to set any environment variables.

### Using a Deployed Scheduler and Taskhub in Azure

Local development with a deployed scheduler:

1. Install the durable task scheduler CLI extension:

    ```bash
    az upgrade
    az extension add --name durabletask --allow-preview true
    ```

1. Create a resource group in a region where the Durable Task Scheduler is available:

    ```bash
    az provider show --namespace Microsoft.DurableTask --query "resourceTypes[?resourceType=='schedulers'].locations | [0]" --out table
    ```

    ```bash
    az group create --name my-resource-group --location <location>
    ```
1. Create a durable task scheduler resource:

    ```bash
    az durabletask scheduler create \
        --resource-group my-resource-group \
        --name my-scheduler \
        --ip-allowlist '["0.0.0.0/0"]' \
        --sku-name "Dedicated" \
        --sku-capacity 1 \
        --tags "{'myattribute':'myvalue'}"
    ```

1. Create a task hub within the scheduler resource:

    ```bash
    az durabletask taskhub create \
        --resource-group my-resource-group \
        --scheduler-name my-scheduler \
        --name "my-taskhub"
    ```

1. Grant the current user permission to connect to the `my-taskhub` task hub:

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

1.  If you're using a deployed scheduler, you need set Environment Variables:
  ```bash
  export ENDPOINT=$(az durabletask scheduler show \
      --resource-group my-resource-group \
      --name my-scheduler \
      --query "properties.endpoint" \
      --output tsv)

  export TASKHUB="my-taskhub"
  ```

1. Install the required packages:
   ```bash
   pip install -r requirements.txt
   ```

1. Start the worker in a terminal:
   ```bash
   python worker.py
   ```
   You should see output indicating the worker has started and registered the entity and orchestration.

1. In a new terminal (with the virtual environment activated if applicable), run the client:
  > **Note:** Remember to set the environment variables again if you're using a deployed scheduler. 

   ```bash
   python client.py [entity-key]
   ```
   You can optionally provide an entity key as an argument. If not provided, "my-counter" will be used.

## Understanding Durable Entities

### Entity Definition

Entities are defined as functions that receive an `EntityContext`:

```python
def counter(ctx: entities.EntityContext, input: int):
    state = ctx.get_state(int, 0)  # Get current state with default
    
    if ctx.operation == "add":
        state += input
        ctx.set_state(state)
    elif ctx.operation == "get":
        return state
```

### Entity Operations

Entities support two types of operations:

1. **Signal (fire-and-forget)**: Sends a message to the entity without waiting for a response
   ```python
   ctx.signal_entity(entity_id=entity_id, operation_name="add", input=10)
   ```

2. **Call (request-response)**: Sends a message and waits for the result
   ```python
   value = yield ctx.call_entity(entity=entity_id, operation="get")
   ```

### Client Operations

Entities can also be signaled directly from clients:
```python
entity_id = entities.EntityInstanceId("counter", "my-counter")
client.signal_entity(entity_id, "add", input=100)
```

## Deploying with Azure Developer CLI (AZD)

This sample includes an `azure.yaml` configuration file that allows you to deploy the entire solution to Azure using Azure Developer CLI (AZD).

> **Note:** This sample uses the shared infrastructure templates located at [`samples/infra/`](../../../infra/).

### Prerequisites for AZD Deployment

1. Install [Azure Developer CLI](https://learn.microsoft.com/en-us/azure/developer/azure-developer-cli/install-azd)
2. Authenticate with Azure:
   ```bash
   azd auth login
   ```

### Deployment Steps

1. Navigate to the Entities sample directory:
   ```bash
   cd /path/to/Durable-Task-Scheduler/samples/durable-task-sdks/python/entities
   ```

2. Initialize the Azure Developer CLI project (only needed the first time):
   ```bash
   azd init
   ```
   This step prepares the environment for deployment and creates necessary configuration files.

3. Provision resources and deploy the application:
   ```bash
   azd up
   ```
   This command will:
   - Provision Azure resources (including Azure Container Apps and Durable Task Scheduler)
   - Build and deploy both the Client and Worker components
   - Set up the necessary connections between components

3. After deployment completes, AZD will display URLs for your deployed services.

4. Monitor your entities and orchestrations using the Azure Portal by navigating to your Durable Task Scheduler resource.

5. To confirm the sample is working correctly, view the application logs through the Azure Portal:
   - Navigate to the Azure Portal (https://portal.azure.com)
   - Go to your resource group where the application was deployed
   - Find and select the Container Apps for both the worker and client components
   - For each Container App:
     - Click on "Log stream" in the left navigation menu under "Monitoring"
     - View the real-time logs showing entity operations and orchestration results

## Understanding the Output

When you run the sample, you'll see output from both the worker and client processes:

### Worker Output
The worker shows:
- Registration of the counter entity and orchestrator
- Log entries when entity operations are performed
- The state changes for each counter entity

### Client Output
The client shows:
- Direct entity signals sent to the counter
- Orchestration scheduling and completion
- Final counter values returned from entity operations

Example output:
```
Starting entity operations demo - 5 orchestrations
=== Direct Entity Operations ===
Signaling entity 'my-counter' to add 100
Signaling entity 'my-counter' to subtract 25
=== Orchestration-based Entity Operations ===
Scheduling orchestration #1 for entity 'my-counter-orch-1'
Orchestration completed successfully with result: "Counter 'my-counter-orch-1' final value: 12"
```

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your entities:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see orchestration instances in the list
4. Click on an instance ID to view the execution details, which will show:
   - Entity signals and calls
   - Entity state changes
   - The final result

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details and entity interactions

The dashboard helps visualize how entities maintain state across multiple operations and orchestrations.
