# Async HTTP API Pattern

## Description of the Sample

This sample demonstrates a mini application that uses Azure Durable Task Scheduler behind the scenes to power asynchronous HTTP APIs. Unlike other samples that focus on specific DurableTask concepts, this example shows how to build a production-ready web application that leverages DTS internally to manage long-running operations.

The application demonstrates:
1. A FastAPI web server exposing RESTful endpoints for managing long-running tasks
2. How to implement the asynchronous operation pattern for HTTP APIs using DTS as the backend infrastructure
3. Integration between a modern web framework and the durable orchestration engine

In this sample:
1. A FastAPI web server exposes endpoints to start operations and check their status
2. When a client requests an operation, an orchestration is started to handle the long-running work
3. The client receives an immediate response with an operation ID and a status endpoint URL
4. The client can poll the status endpoint to check when the operation completes
5. The long-running operation is simulated by the `process_long_running_operation` activity

This pattern is useful for:
- Exposing long-running operations via HTTP APIs
- Implementing the REST asynchronous operation pattern
- Building responsive web APIs that handle operations taking longer than a typical HTTP request timeout
- Providing status tracking for operations that might take seconds, minutes, or even hours

## Prerequisites

1. [Python 3.9+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator) installed
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)
4. [FastAPI](https://fastapi.tiangolo.com/) and [Uvicorn](https://www.uvicorn.org/) (installed via requirements.txt)

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

1. In a new terminal (with the virtual environment activated if applicable), run the client (which is a FastAPI server):
  > **Note:** Remember to set the environment variables again if you're using a deployed scheduler. 

  ```bash
  python client.py
  ```
  This will start a FastAPI server on port 8000.

1. Interact with the API using a browser, curl, or PowerShell:
   
   **Using curl:**
   - To start a new operation:
     ```bash
     curl -X POST http://localhost:8000/api/start-operation -H "Content-Type: application/json" -d '{"processing_time": 10}'
     ```
   - To check operation status (replace `{operation_id}` with the ID from the previous response):
     ```bash
     curl http://localhost:8000/api/operations/{operation_id}
     ```
   
   **Using PowerShell:**
   - To start a new operation:
     ```powershell
     Invoke-RestMethod -Method Post -Uri "http://localhost:8000/api/start-operation" `
     -Headers @{ "Content-Type" = "application/json" } `
     -Body '{"processing_time": 10}'
     ```
   - To check operation status (replace `{operation_id}` with the ID from the previous response):
     ```powershell
     Invoke-RestMethod -Uri "http://localhost:8000/api/operations/{operation_id}"
     ```

## Identity-based authentication

Learn how to set up [identity-based authentication](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-identity?tabs=df&pivots=az-cli) when you deploy the app Azure.  

## Understanding the Output

When you run the sample, you'll see output from both the worker and client processes:

### Worker Output
The worker shows:
- Registration of the orchestrator and activities
- Log entries when long-running operations are being processed
- Information about each operation including its ID and processing time
- Completion messages when operations finish

### Client Output (FastAPI Server)
The client (FastAPI server) shows:
- Server startup information
- Log entries for API requests received
- Starting operations when POST requests are made
- Status checks when GET requests are made

### API Response Examples

When starting an operation:
```json
{
  "operation_id": "3f7b8ac2-5e6d-4f3g-9h2i-1j2k3l4m5n6o",
  "status_url": "/api/operations/3f7b8ac2-5e6d-4f3g-9h2i-1j2k3l4m5n6o"
}
```

When checking status (in progress):
```json
{
  "operation_id": "3f7b8ac2-5e6d-4f3g-9h2i-1j2k3l4m5n6o",
  "status": "RUNNING",
  "last_updated": "2023-05-10T15:30:45.123456Z"
}
```

When checking status (completed):
```json
{
  "operation_id": "3f7b8ac2-5e6d-4f3g-9h2i-1j2k3l4m5n6o",
  "status": "Completed",
  "result": {
    "operation_id": "3f7b8ac2-5e6d-4f3g-9h2i-1j2k3l4m5n6o",
    "status": "completed",
    "result": "Operation 3f7b8ac2-5e6d-4f3g-9h2i-1j2k3l4m5n6o completed successfully",
    "processed_at": 1683737445.123456
  }
}
```

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestration:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see the orchestration instance(s) in the list
4. Click on an instance ID to view the execution details, which will show:
   - The call to the `process_long_running_operation` activity
   - The input parameters including operation ID and processing time
   - The completed result with timing information

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details

The dashboard helps you understand how the async HTTP API pattern works behind the scenes, showing how the durable orchestration provides the backend processing for the asynchronous API endpoints.
