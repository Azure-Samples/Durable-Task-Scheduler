# Eternal Orchestrations Pattern

## Description of the Sample

This sample demonstrates the eternal orchestrations pattern with Azure Durable Functions in Python. This pattern is designed for creating long-running, recurring workflows that can continue indefinitely until explicitly stopped, making it ideal for periodic tasks, monitoring jobs, and scheduled processes.

In this sample:
1. **Initialization**: The orchestration starts with specified parameters (task type, interval, max iterations)
2. **Work Execution**: The `perform_periodic_work` activity executes the recurring task
3. **Timer Wait**: Uses a durable timer to wait for the next execution interval without consuming resources  
4. **Self-Continuation**: The orchestration continues itself with `continue_as_new()` to reset the execution history and prevent memory buildup
5. **Graceful Termination**: Can be stopped via external events or after reaching maximum iterations

This pattern is useful for:
- Periodic data synchronization or ETL jobs
- Health monitoring and system status checks
- Regular cleanup or maintenance tasks
- Scheduled reporting or notifications
- Long-running processes that need to execute at regular intervals

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

4. Start an eternal orchestration by sending a POST request:
   ```bash
   # Start with default parameters (2 minute intervals, 5 iterations max)
   curl -X POST http://localhost:7071/api/eternal_orchestration \
     -H "Content-Type: application/json" \
     -d '{}'

   # Start with custom parameters
   curl -X POST http://localhost:7071/api/eternal_orchestration \
     -H "Content-Type: application/json" \
     -d '{"task_type": "data_sync", "interval_minutes": 1, "max_iterations": 10, "target_url": "https://httpbin.org/delay/1"}'
   ```

5. Monitor the running orchestration:
   ```bash
   curl http://localhost:7071/api/status/{instanceId}
   ```

6. Stop the eternal orchestration when needed:
   ```bash
   curl -X POST http://localhost:7071/api/stop/{instanceId}
   ```

## Understanding the Output

When you run the sample, you'll see the following behavior:

1. **Initial Response**: The HTTP trigger returns management URLs immediately:
   ```json
   {
     "id": "abcd1234",
     "statusQueryGetUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234",
     "sendEventPostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234/raiseEvent/{eventName}",
     "terminatePostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234/terminate?reason={text}",
     "purgeHistoryDeleteUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234"
   }
   ```

2. **Running Status**: While the eternal orchestration is active, status checks will show:
   ```json
   {
     "runtimeStatus": "Running",
     "input": {"task_type": "health_check", "interval_minutes": 2, "max_iterations": 5},
     "customStatus": {
       "current_iteration": 3,
       "max_iterations": 5,
       "last_run": "2025-09-19T16:45:30Z",
       "task_type": "health_check"
     }
   }
   ```

3. **Completed Status**: When reaching maximum iterations or being stopped:
   ```json
   {
     "runtimeStatus": "Completed", 
     "output": {
       "status": "completed",
       "total_iterations": 5,
       "final_result": {
         "iteration": 5,
         "task_type": "health_check",
         "success": true,
         "executed_at": "2025-09-19T16:45:30Z"
       }
     }
   }
   ```

4. **Key Characteristics**:
   - **Resource Efficient**: Uses durable timers instead of blocking waits between iterations
   - **Self-Continuation**: Calls `continue_as_new()` to reset execution history and prevent memory buildup
   - **Graceful Shutdown**: Responds to external stop events or completion conditions
   - **Long-Term Stability**: Designed to run for extended periods (days, weeks, or months)
   - **Progress Tracking**: Maintains iteration counts and execution history via custom status

## Dashboard Review

You can monitor the orchestration execution through the Durable Task Scheduler dashboard:

1. Navigate to `http://localhost:8082` in your browser
2. You'll see a list of task hubs - select the "default" hub
3. Click on your orchestration instance to see:
   - The eternal orchestration running continuously with regular activity executions
   - Durable timers showing scheduled next execution times between iterations
   - Custom status updates tracking current iteration count and progress  
   - Timeline showing the recurring pattern of work activity followed by timer wait
   - How `continue_as_new()` resets the execution history while maintaining the iteration loop

The dashboard is particularly valuable for this pattern because it demonstrates how eternal orchestrations can run indefinitely while managing resources efficiently through self-continuation and durable timers.

## Learn More

- [Eternal Orchestrations Pattern in Durable Functions](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-eternal-orchestrations)
- [Durable Task Scheduler Overview](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
- [Durable Functions Python Developer Guide](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-python)