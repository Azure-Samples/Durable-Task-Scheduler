# Monitoring Pattern

## Description of the Sample

This sample demonstrates the monitoring pattern with the Azure Durable Task Scheduler using the Python SDK. The monitoring pattern is used for periodically checking the status of a long-running operation until it completes or times out.

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
   You should see output indicating the worker has started and registered the orchestration and activities.

1. In a new terminal (with the virtual environment activated if applicable), run the client:
  > **Note:** Remember to set the environment variables again if you're using a deployed scheduler. 
  
   ```bash
   python client.py [job_id] [polling_interval] [timeout]
   ```
   You can optionally provide:
   - `job_id`: A unique identifier for the job (defaults to a random UUID)
   - `polling_interval`: How often to check job status in seconds (defaults to 5)
   - `timeout`: Maximum monitoring duration in seconds (defaults to 30)

## Identity-based authentication

Learn how to set up [identity-based authentication](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-identity?tabs=df&pivots=az-cli) when you deploy the app Azure.  


## Understanding the Output

When you run the sample, you'll see output from both the worker and client processes:

### Worker Output
The worker shows:
- Registration of the orchestrator and activities
- Starting the monitoring orchestration with the specified parameters
- Periodic log entries when the `check_job_status` activity is called
- Status updates as the check count increases
- A message when monitoring completes or times out

### Client Output
The client shows:
- Starting the monitoring orchestration for the specified job
- Real-time status updates as they occur (via custom status)
- Status changes from "Unknown" to "Running" and finally to "Completed"
- The final result, including:
  - Job ID
  - Final status
  - Number of status checks performed
  - Total monitoring duration

The sample demonstrates a job that completes after 3 status checks, but in a real application, the `check_job_status` activity would typically call an external service to determine the actual status.

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestration:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see the orchestration instance in the list
4. Click on the instance ID to view the execution details, which will show:
   - The periodic calls to the `check_job_status` activity
   - The timers created between checks
   - The custom status updates at each step
   - The final result when monitoring completes

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details

The dashboard visualizes the polling pattern, showing how the orchestrator alternates between checking status and waiting, and how it uses timers to implement the polling interval.
