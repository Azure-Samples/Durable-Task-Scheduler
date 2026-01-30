# Orchestration Versioning Pattern

## Description of the Sample

This sample demonstrates the Orchestration Versioning pattern with the Azure Durable Task Scheduler using the Python SDK. Versioning allows you to safely evolve orchestration logic while maintaining backward compatibility for in-flight orchestrations.

In this sample:
1. A versioned orchestration is defined that changes behavior based on its version
2. The client schedules orchestrations with different version strings (1.0.0, 2.0.0, 3.0.0)
3. The orchestration uses `ctx.version` to branch logic based on the version
4. All versions run on the same worker, ensuring backward compatibility

**Version history in this sample:**
- **v1.0.0**: Basic hello greeting
- **v2.0.0**: Added goodbye greeting
- **v3.0.0**: Added notification after greeting

This pattern is useful for:
- Safely deploying new orchestration logic without breaking in-flight workflows
- Maintaining multiple versions of business logic simultaneously
- Gradual rollout of new features
- Avoiding non-deterministic errors during orchestration replay

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
   You should see output indicating the worker has started and registered the activities and orchestration.

1. In a new terminal (with the virtual environment activated if applicable), run the client:
  > **Note:** Remember to set the environment variables again if you're using a deployed scheduler.

   ```bash
   python client.py [name]
   ```
   You can optionally provide a name as an argument. If not provided, "World" will be used.

## Understanding Orchestration Versioning

### Setting the Version

When scheduling an orchestration, the client specifies the version:

```python
instance_id = client.schedule_new_orchestration(
    "versioned_orchestration",
    input="World",
    version="2.0.0"  # Version is set here
)
```

### Reading the Version in Orchestrations

Inside the orchestration, use `ctx.version` to read the version:

```python
def versioned_orchestration(ctx: task.OrchestrationContext, name: str):
    orch_version = ctx.version  # e.g., "2.0.0"
    
    # Always run v1 logic
    result = yield ctx.call_activity(activity_v1, input=name)
    
    # Only run v2+ logic
    if compare_version(orch_version, "2.0.0") >= 0:
        result = yield ctx.call_activity(activity_v2, input=name)
    
    return result
```

### Version Comparison Helper

The sample includes a helper function for semantic version comparison:

```python
from packaging import version

def compare_version(v1: str | None, v2: str) -> int:
    """Compare two version strings.
    
    Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
    """
    if v1 is None:
        return -1
    try:
        ver1 = version.parse(v1)
        ver2 = version.parse(v2)
        if ver1 < ver2:
            return -1
        elif ver1 > ver2:
            return 1
        return 0
    except Exception:
        # Fall back to string comparison
        return (v1 > v2) - (v1 < v2)
```

### Why Versioning Matters

Without versioning, changing orchestration logic can cause **non-deterministic errors**:

1. An orchestration starts with v1 logic
2. You deploy new code with v2 logic (adds new activity)
3. The orchestration replays but hits the new activity code
4. **ERROR**: History doesn't match the new code path

With versioning:
1. v1 orchestrations continue using v1 code path
2. New orchestrations use v2 code path
3. Both run on the same worker without conflict

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

1. Navigate to the Versioning sample directory:
   ```bash
   cd /path/to/Durable-Task-Scheduler/samples/durable-task-sdks/python/versioning
   ```

2. Initialize the Azure Developer CLI project (only needed the first time):
   ```bash
   azd init
   ```

3. Provision resources and deploy the application:
   ```bash
   azd up
   ```

4. After deployment completes, AZD will display URLs for your deployed services.

5. Monitor your orchestrations using the Azure Portal by navigating to your Durable Task Scheduler resource.

## Understanding the Output

When you run the sample, you'll see output from both the worker and client processes:

### Worker Output
The worker shows:
- Registration of activities and orchestrator
- Log entries when activities are called
- Which activities are called varies by version

### Client Output
The client shows three orchestrations with different versions:

```
=== Orchestration Versioning Demo ===
Testing with name: World

Scheduling orchestration with version 1.0.0: v1 - Basic hello only
  Instance ID: abc123...
Scheduling orchestration with version 2.0.0: v2 - Hello + Goodbye
  Instance ID: def456...
Scheduling orchestration with version 3.0.0: v3 - Hello + Goodbye + Notification
  Instance ID: ghi789...

Waiting for orchestrations to complete...

=== Version 1.0.0 (v1 - Basic hello only) ===
  Status: COMPLETED
  Result: {"version": "1.0.0", "results": ["Hello, World!"]}

=== Version 2.0.0 (v2 - Hello + Goodbye) ===
  Status: COMPLETED
  Result: {"version": "2.0.0", "results": ["Hello, World!", "Goodbye, World!"]}

=== Version 3.0.0 (v3 - Hello + Goodbye + Notification) ===
  Status: COMPLETED
  Result: {"version": "3.0.0", "results": ["Hello, World!", "Goodbye, World!", "Notification sent: ..."]}

=== Demo Complete ===
Key takeaway: All versions ran using the same worker code!
```

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestrations:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see orchestration instances in the list
4. Click on an instance ID to view the execution details
5. Notice how different versions have different numbers of activity calls

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details

The dashboard helps visualize how different versions execute different code paths while using the same orchestration definition.
