# Async HTTP API Pattern

## Description of the Sample

This sample demonstrates the async HTTP API pattern with Azure Durable Functions in Python. This pattern is essential for handling long-running operations where clients need to periodically check the status of their requests rather than waiting for a synchronous response.

In this sample:
1. **HTTP Request**: Client submits a request to start a long-running operation
2. **Immediate Response**: The HTTP trigger returns immediately with status URLs and a 202 Accepted status
3. **Background Processing**: The orchestrator manages a long-running `process_long_running_task` activity
4. **Status Polling**: Clients use the provided URLs to check the operation status
5. **Completion**: Eventually, the operation completes and returns the final result

This pattern is useful for:
- Operations that take several seconds or minutes to complete
- Batch processing jobs where clients need to track progress
- Integration scenarios where you need to prevent HTTP timeouts
- APIs that need to provide immediate responses for long-running tasks

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

4. Start a long-running operation by sending a POST request:
   ```bash
   # Start operation with default parameters
   curl -X POST http://localhost:7071/api/async_http_api \
     -H "Content-Type: application/json" \
     -d '{}'

   # Start operation with custom parameters
   curl -X POST http://localhost:7071/api/async_http_api \
     -H "Content-Type: application/json" \
     -d '{"operation_type": "data_processing", "duration": 45, "data": {"input": "sample"}}'
   ```

5. Poll for status using the URLs from the initial response:
   ```bash
   # Check status using the statusQueryGetUri from the response
   curl http://localhost:7071/api/status/{instanceId}
   ```

## Understanding the Output

When you run the sample, you'll see the following behavior:

1. **Initial HTTP Response**: The operation starts immediately and returns HTTP 202 (Accepted) with management URLs:
   ```json
   {
     "id": "abcd1234",
     "statusQueryGetUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234",
     "sendEventPostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234/raiseEvent/{eventName}",
     "terminatePostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234/terminate?reason={text}",
     "purgeHistoryDeleteUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234"
   }
   ```

2. **Status Polling**: While the long-running task executes, status checks will show:
   ```json
   {
     "runtimeStatus": "Running",
     "input": {"operation_type": "data_processing", "duration": 30},
     "output": null,
     "createdTime": "2023-12-01T10:00:00Z",
     "lastUpdatedTime": "2023-12-01T10:00:15Z"
   }
   ```

3. **Completion**: Once the operation finishes (after the specified duration), the status will show:
   ```json
   {
     "runtimeStatus": "Completed",
     "input": {"operation_type": "data_processing", "duration": 30},
     "output": {
       "task": "data_processing",
       "result": "Success",
       "duration": 30,
       "timestamp": "2023-12-01T10:00:45Z"
     }
   }
   ```

4. **Key Benefits**: This pattern allows clients to:
   - Get immediate confirmation that the request was accepted
   - Avoid HTTP timeouts on long-running operations  
   - Check progress at their own pace
   - Handle other tasks while waiting for completion

## Dashboard Review

You can monitor the orchestration execution through the Durable Task Scheduler dashboard:

1. Navigate to `http://localhost:8082` in your browser
2. You'll see a list of task hubs - select the "default" hub
3. Click on your orchestration instance to see:
   - Real-time status updates as the long-running task progresses
   - Timeline showing when the operation started and how long it's been running
   - Input parameters and current execution state
   - The async HTTP API pattern in action with clear start/processing/completion phases

The dashboard is particularly valuable for this pattern because it demonstrates how the orchestration continues running in the background while clients can poll for updates through the HTTP API.

## Learn More

- [Async HTTP APIs Pattern in Durable Functions](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-http-features)
- [Durable Task Scheduler Overview](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
- [Durable Functions Python Developer Guide](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-python)