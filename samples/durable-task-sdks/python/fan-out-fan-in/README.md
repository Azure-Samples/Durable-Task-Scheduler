# Fan Out/Fan In Pattern

## Description of the Sample

This sample demonstrates the fan out/fan in pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern is used for parallel processing of multiple items, followed by an aggregation of the results.

In this sample:
1. The orchestrator receives a list of work items as input
2. It "fans out" by creating parallel tasks for each work item (calling `process_work_item` for each one)
3. It waits for all tasks to complete using `task.when_all`
4. It then "fans in" by aggregating the results with the `aggregate_results` activity
5. The final aggregated result is returned to the client

This pattern is useful for:
- Processing multiple items concurrently to improve throughput
- Performing calculations on batches of data
- Running operations in parallel that don't depend on each other
- Aggregating results from multiple parallel operations

## Prerequisites

1. [Python 3.9+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
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
   python client.py [number_of_items]
   ```
   You can optionally provide the number of work items as an argument. If not provided, 10 items will be used by default.

## Understanding the Output

When you run the sample, you'll see output from both the worker and client processes:

### Worker Output
The worker shows:
- Registration of the orchestrator and activities
- Status messages when processing each work item in parallel, showing that they're executing concurrently
- Random delays for each work item (between 0.5 and 2 seconds) to simulate varying processing times
- A final message showing the aggregation of results

### Client Output
The client shows:
- Starting the orchestration with the specified number of work items
- The unique orchestration instance ID
- The final aggregated result, which includes:
  - Total number of items processed
  - Sum of all results (each item result is the square of its value)
  - Average of all results

The example demonstrates how multiple items can be processed in parallel, with the results gathered and aggregated once all parallel tasks are complete.

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestration:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see the orchestration instance in the list
4. Click on the instance ID to view the execution details, which will show:
   - The parallel execution of multiple `process_work_item` activities
   - The wait for all tasks to complete using `task.when_all`
   - The final call to `aggregate_results` with the collected results
   - The inputs and outputs for each activity

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details

The dashboard visualizes the concurrent execution of the tasks, allowing you to see how the fan-out/fan-in pattern improves throughput by processing items in parallel.
