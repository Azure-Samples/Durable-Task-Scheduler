# Eternal Orchestrations Pattern

## Description of the Sample

This sample demonstrates the eternal orchestrations pattern with the Azure Durable Task Scheduler using the Python SDK. In this pattern, an orchestration function runs continuously by recreating itself at the end of its execution using the `continue_as_new` method.

In this sample:
1. The `periodic_cleanup` orchestration function calls a cleanup activity
2. It then creates a timer that waits for 15 seconds
3. After the timer expires, it calls `continue_as_new` with an incremented counter
4. This process repeats for a specified number of iterations (5 in this case)

Eternal orchestrations are useful for recurring tasks, monitoring processes, or any workflow that needs to run for an extended period without accumulating execution history.

## Prerequisites

1. [Python 3.9+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator) installed
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample:

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

For production scenarios or when you're ready to deploy to Azure:

1. Find regions that support the Durable Task Scheduler: 
  ```bash
  az provider show --namespace Microsoft.DurableTask --query "resourceTypes[?resourceType=='schedulers'].locations | [0]" --out table
  ```

1. Create a Scheduler using the Azure CLI:
  ```bash
  az durabletask scheduler create --resource-group <resource-group> --name <scheduler-name> --location <location> --ip-allowlist "[0.0.0.0/0]" --sku-capacity 1 --sku-name "Dedicated" --tags "{'myattribute':'myvalue'}"
  ```

1. Create Your Taskhub:
  ```bash
  az durabletask taskhub create --resource-group <resource-group> --scheduler-name <scheduler-name> --name <taskhub-name>
  ```

1. Assign your identity access to the task hub: 
    ```bash
    assignee=$(az ad user show --id "someone@microsoft.com" --query "id" --output tsv)

    scope="/subscriptions/<subscription-id>/resourceGroups/<resource-group>/providers/Microsoft.DurableTask/schedulers/<scheduler-name>/taskHubs/<taskhub-name>"

    az role assignment create --assignee "$assignee" --role "Durable Task Data Contributor" --scope "$scope"
    ```

## How to Run the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. First, activate your Python virtual environment (if you're using one):
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows, use: venv\Scripts\activate
   ```

1.  If you're using a deployed scheduler, you need set Environment Variables. Note the scheduler endpoint can be found in the Scheduler's overview tab on Azure Portal.
   ```bash
   export TASKHUB=<taskhubname>
   export ENDPOINT=<schedulerEndpoint>
   ```

1. Install the required packages:
   ```bash
   pip install -r requirements.txt
   ```

1. Start the worker in a terminal:
   ```bash
   python worker.py
   ```
   You should see output indicating the worker has started and registered the orchestration and activities.

1. In a new terminal (with the virtual environment activated if applicable), run the client:
  > **Note:** Remember to set the environment variables again if you're using a deployed scheduler. 

   ```bash
   python client.py
   ```

## Understanding the Output

When you run the sample, you'll see output from both the worker and client processes:

### Worker Output
The worker shows:
- Registration of the orchestration and activity
- Cleanup activity being called approximately every 15 seconds
- Messages indicating when the cleanup activity is running
- Each new iteration of the orchestration after the `continue_as_new` call

### Client Output
The client shows:
- Starting of the new orchestration instance with an initial counter value
- The orchestration ID of the started instance

Example output:
```
Starting Eternal Orchestrations pattern client...
Using taskhub: default
Using endpoint: http://localhost:8080
Started eternal orchestration with ID = <instance-id>
```

Unlike other examples, you won't see a completion message because the orchestration is designed to run continuously for a set number of iterations. In this sample, it will run for 5 iterations before stopping.

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestration:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see the orchestration instance in the list
4. If you click on the instance ID, you'll notice something interesting:
   - You'll see only the most recent iteration of the orchestration
   - Previous execution history is replaced each time `continue_as_new` is called
   - This is a key benefit of eternal orchestrations - they avoid unbounded history growth

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details

The dashboard helps visualize how eternal orchestrations work by showing only the current iteration's execution history. This is particularly important for long-running orchestrations as it prevents the history from growing indefinitely, which would impact performance and storage.
