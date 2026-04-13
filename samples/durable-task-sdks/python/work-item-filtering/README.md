# Work Item Filtering

Python | Durable Task SDK

## Description of the Sample

This sample demonstrates work item filtering with the Azure Durable Task Scheduler using the Python SDK. Work item filtering allows you to run multiple specialized workers where each worker only processes specific orchestrations and activities, enabling workload isolation and independent scaling.

In this sample:
1. **Worker A** registers a `greeting_orchestrator` with a `say_hello` activity and enables auto-generated filters
2. **Worker B** registers a `math_orchestrator` with an `add_numbers` activity and enables auto-generated filters
3. The client schedules both types of orchestrations
4. Each orchestration is routed only to the worker that registered the matching orchestrator and activities

This pattern is useful for:
- Running specialized workers that handle different workload types
- Scaling workers independently based on workload characteristics (e.g., CPU-intensive vs. I/O-bound)
- Isolating workloads for reliability — a failure in one worker type doesn't affect others
- Deploying updates to specific orchestration types without affecting the rest

## Prerequisites

1. [Python 3.10+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

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

Note: The example code automatically uses the default emulator settings (endpoint: `http://localhost:8080`, taskhub: `default`). You don't need to set any environment variables.

### Using a Deployed Scheduler and Taskhub in Azure

Local development with a deployed scheduler:

1. Install the durable task scheduler CLI extension:

    ```bash
    az upgrade
    az extension add --name durabletask --allow-preview true
    ```

2. Create a resource group in a region where the Durable Task Scheduler is available:

    ```bash
    az provider show --namespace Microsoft.DurableTask --query "resourceTypes[?resourceType=='schedulers'].locations | [0]" --out table
    ```

    ```bash
    az group create --name my-resource-group --location <location>
    ```

3. Create a durable task scheduler resource:

    ```bash
    az durabletask scheduler create \
        --resource-group my-resource-group \
        --name my-scheduler \
        --ip-allowlist '["0.0.0.0/0"]' \
        --sku-name "Dedicated" \
        --sku-capacity 1 \
        --tags "{'myattribute':'myvalue'}"
    ```

4. Create a task hub within the scheduler resource:

    ```bash
    az durabletask taskhub create \
        --resource-group my-resource-group \
        --scheduler-name my-scheduler \
        --name "my-taskhub"
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

## How to Run the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. First, activate your Python virtual environment (if you're using one):
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows, use: venv\Scripts\activate
   ```

2. If you're using a deployed scheduler, set environment variables:
   ```bash
   export ENDPOINT=$(az durabletask scheduler show \
       --resource-group my-resource-group \
       --name my-scheduler \
       --query "properties.endpoint" \
       --output tsv)

   export TASKHUB="my-taskhub"
   ```

3. Install the required packages:
   ```bash
   pip install -r requirements.txt
   ```

4. Start Worker A (greeting worker) in a terminal:
   ```bash
   python worker_a.py
   ```

5. Start Worker B (math worker) in a second terminal:
   ```bash
   python worker_b.py
   ```

6. In a third terminal, run the client:
   > **Note:** Remember to set the environment variables again if you're using a deployed scheduler.

   ```bash
   python client.py
   ```

## Expected Output

### Worker A Output
```
[Worker A] Using taskhub: default
[Worker A] Using endpoint: http://localhost:8080
INFO:__main__:[Worker A] Ready — processing only greeting orchestrations
INFO:__main__:[Worker A] say_hello called with name: World
```

### Worker B Output
```
[Worker B] Using taskhub: default
[Worker B] Using endpoint: http://localhost:8080
INFO:__main__:[Worker B] Ready — processing only math orchestrations
INFO:__main__:[Worker B] add_numbers called with a=40, b=2
```

### Client Output
```
--- Scheduling greeting orchestration (Worker A) ---
--- Scheduling math orchestration (Worker B) ---

Waiting for orchestrations to complete...
Greeting result: "Hello, World!"
Math result: 42

Done! Check the worker terminal outputs to see which worker handled each orchestration.
```

Notice that Worker A only processed the greeting orchestration and Worker B only processed the math orchestration.

## Code Walkthrough

### Auto-Generated Filters

The simplest way to enable filtering is to call `use_work_item_filters()` with no arguments after registering your orchestrators and activities:

```python
worker.add_orchestrator(greeting_orchestrator)
worker.add_activity(say_hello)
worker.use_work_item_filters()  # Auto-generate from registry
```

The SDK automatically builds filters from everything registered with the worker.

### Explicit Filters

For more control, you can pass explicit `WorkItemFilters` with optional version constraints:

```python
from durabletask import worker

worker.use_work_item_filters(worker.WorkItemFilters(
    orchestrations=[
        worker.OrchestrationWorkItemFilter(name="greeting_orchestrator", versions=["1.0"]),
    ],
    activities=[
        worker.ActivityWorkItemFilter(name="say_hello"),
    ],
))
```

## Viewing in the Dashboard

- **Emulator:** Navigate to http://localhost:8082 → select the "default" task hub
- **Azure:** Navigate to your Scheduler resource in the Azure Portal → Task Hub → Dashboard URL

## Related Samples

- [Function Chaining](../function-chaining/) - Basic sequential workflow pattern
- [Orchestration Versioning](../versioning/) - Safe orchestration evolution with version constraints

## Learn More

- [Durable Task Scheduler documentation](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/develop-with-durable-task-scheduler)
