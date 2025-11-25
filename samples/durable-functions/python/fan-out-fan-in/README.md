# Fan-Out/Fan-In Pattern

## Description of the Sample

This sample demonstrates the fan-out/fan-in pattern with Azure Durable Functions in Python. This pattern is useful for executing multiple activities in parallel and then aggregating their results when all activities complete.

In this sample:
1. **Fan-Out**: The orchestrator starts multiple `process_work_item` activities in parallel, one for each work item
2. **Parallel Processing**: Each activity processes its work item independently and concurrently
3. **Fan-In**: The orchestrator waits for all activities to complete and collects their results
4. **Aggregation**: An `aggregate_results` activity combines all results into a summary report

This pattern is useful for:
- Processing large datasets by breaking them into chunks
- Performing parallel calculations or transformations
- Distributing workload across multiple workers for better performance
- Scenarios where independent tasks can be executed simultaneously

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure Functions Core Tools](https://docs.microsoft.com/en-us/azure/azure-functions/functions-run-local) v4.x
3. [Docker](https://www.docker.com/products/docker-desktop/) (for running the Durable Task Scheduler) installed

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

Once you have set up the Durable Task Scheduler, follow these steps to run the sample:

1. First, activate your Python virtual environment (if you're using one):
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows, use: venv\Scripts\activate
   ```

2. Install the required packages:
   ```bash
   pip install -r requirements.txt
   ```

3. Start the Azure Functions runtime:
   ```bash
   func start
   ```
   
   You should see output indicating the functions have loaded successfully.

4. Test the orchestration by sending a POST request:
   ```bash
   # Start orchestration with default work items
   curl -X POST http://localhost:7071/api/fan_out_fan_in \
     -H "Content-Type: application/json" \
     -d '{}'

   # Start orchestration with custom work items
   curl -X POST http://localhost:7071/api/fan_out_fan_in \
     -H "Content-Type: application/json" \
     -d '{"workItems": ["Task1", "Task2", "Task3", "Task4", "Task5", "Task6"]}'
   ```

5. Check orchestration status using the `statusQueryGetUri` from the response:
   ```bash
   curl http://localhost:7071/api/status/{instanceId}
   ```

## Understanding the Output

When you run the sample, you'll see the following behavior:

1. **Initial Response**: The HTTP trigger returns a JSON response with management URLs:
   ```json
   {
     "id": "abcd1234",
     "statusQueryGetUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234",
     "sendEventPostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234/raiseEvent/{eventName}",
     "terminatePostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234/terminate?reason={text}",
     "purgeHistoryDeleteUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234"
   }
   ```

2. **Parallel Processing**: The orchestrator starts multiple `process_work_item` activities simultaneously. Each activity:
   - Processes its work item independently
   - Simulates work with random processing times (0.5-2.0 seconds)
   - Returns processing metrics and a random value

3. **Aggregated Results**: For input `{"workItems": ["Task1", "Task2", "Task3"]}`, the final output will be:
   ```json
   {
     "total_items_processed": 3,
     "total_value": 165,
     "average_value": 55.0,
     "total_processing_time": 3.47,
     "processed_items": ["Processed_Task1", "Processed_Task2", "Processed_Task3"],
     "success": true
   }
   ```

4. **Performance Benefits**: Because activities run in parallel rather than sequentially, the total processing time is much shorter than the sum of individual processing times.

## Dashboard Review

You can monitor the orchestration execution through the Durable Task Scheduler dashboard:

1. Navigate to `http://localhost:8082` in your browser
2. You'll see a list of task hubs - select the "default" hub
3. Click on your orchestration instance to see:
   - Parallel activity execution timeline showing concurrent processing
   - Input and output data for each `process_work_item` activity
   - Performance metrics demonstrating the fan-out/fan-in pattern
   - Visual representation of how activities start simultaneously and complete at different times

The dashboard is particularly useful for this sample because it clearly shows how the fan-out/fan-in pattern executes multiple activities concurrently, leading to significant performance improvements compared to sequential processing.

## Learn More

- [Fan-Out/Fan-In Pattern in Durable Functions](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-cloud-backup)
- [Durable Task Scheduler Overview](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
- [Durable Functions Python Developer Guide](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-python)