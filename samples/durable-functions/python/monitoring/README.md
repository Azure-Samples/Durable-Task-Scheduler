# Monitoring Pattern

## Description of the Sample

This sample demonstrates the monitoring pattern with Azure Durable Functions in Python. The monitoring pattern is used for periodically checking the status of a long-running operation until it completes or times out.

In this sample:
1. The orchestrator starts monitoring a job with a specified ID
2. It periodically calls the `check_job_status` activity at defined intervals
3. The current job status is exposed via custom status, making it available to clients
4. Monitoring continues until either:
   - The job completes successfully
   - The specified timeout period is reached

This pattern is useful for:
- Polling external services or APIs that don't support callbacks
- Checking the status of long-running operations
- Implementing timeout mechanisms for operations with unpredictable durations
- Maintaining state about progress of a workflow

## Sample Architecture

```
HTTP Start → Monitoring Orchestrator
                 ├── Check Job Status (Activity)
                 ├── Wait (Timer)
                 ├── Check Job Status (Activity)
                 ├── Wait (Timer)
                 └── ... (repeat until completion or timeout)
```

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

4. Start job monitoring by sending a POST request:
   ```bash
   # Basic job monitoring with default configuration
   curl -X POST http://localhost:7071/api/start_monitoring_job \
     -H "Content-Type: application/json" \
     -d '{}'

   # Job monitoring with custom parameters
   curl -X POST http://localhost:7071/api/start_monitoring_job \
     -H "Content-Type: application/json" \
     -d '{
       "job_id": "my-custom-job-123",
       "polling_interval_seconds": 10,
       "timeout_seconds": 60
     }'
   ```

5. Check orchestration status using the `statusQueryGetUri` from the response:
   ```bash
   curl -X GET "http://localhost:7071/runtime/webhooks/durabletask/instances/{instanceId}"
   ```

6. Optionally, check job status directly:
   ```bash
   curl -X GET "http://localhost:7071/api/job_status/{jobId}"
   ```

## Understanding the Output

When you run the sample, you'll see the following behavior:

1. **Initial Response**: The HTTP trigger returns management URLs immediately:
   ```json
   {
     "id": "abc123def456",
     "statusQueryGetUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abc123def456",
     "sendEventPostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abc123def456/raiseEvent/{eventName}",
     "terminatePostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abc123def456/terminate?reason={text}",
     "purgeHistoryDeleteUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abc123def456"
   }
   ```

2. **Orchestration Status During Monitoring**: While the job is being monitored, status checks will show:
   ```json
   {
     "name": "monitoring_job_orchestrator",
     "instanceId": "abc123def456",
     "runtimeStatus": "Running",
     "input": {
       "job_id": "job-uuid-12345",
       "polling_interval_seconds": 5,
       "timeout_seconds": 30
     },
     "customStatus": {
       "job_id": "job-uuid-12345",
       "status": "Running",
       "check_count": 3,
       "last_check_time": "2025-09-19T18:05:15.123Z"
     }
   }
   ```

3. **Completed Job Monitoring Result**: When the job completes successfully:
   ```json
   {
     "job_id": "job-uuid-12345",
     "final_status": "Completed",
     "checks_performed": 4,
     "monitoring_duration_seconds": 15.6
   }
   ```

4. **Timeout Scenario**: If the job doesn't complete within the specified timeout:
   ```json
   {
     "job_id": "job-uuid-67890",
     "final_status": "Timeout",
     "checks_performed": 6,
     "monitoring_duration_seconds": 30.0
   }
   ```

5. **Monitoring Pattern Benefits**:
   - Provides real-time visibility into job progress via custom status
   - Prevents infinite waiting with built-in timeout handling
   - Uses configurable polling intervals to balance responsiveness and resource usage
   - Handles external services that don't support callbacks or webhooks

## Dashboard Review

You can monitor the orchestration execution through the Durable Task Scheduler dashboard:

1. Navigate to `http://localhost:8082` in your browser
2. You'll see a list of task hubs - select the "default" hub  
3. Click on your orchestration instance to see:
   - Real-time custom status updates showing job monitoring progress
   - Timeline of periodic `check_job_status` activity executions
   - Activity-based delays between status checks (using `wait_for_interval`)
   - How the monitoring pattern continues until job completion or timeout

The dashboard is particularly useful for this pattern because it shows how the orchestration maintains state between periodic checks, demonstrating the monitoring pattern's ability to track long-running external operations over time.

## Learn More

- [Monitoring Pattern in Durable Functions](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-monitor)
- [Durable Task Scheduler Overview](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
- [Durable Functions Python Developer Guide](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-python)