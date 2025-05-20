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
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)
4. [FastAPI](https://fastapi.tiangolo.com/) and [Uvicorn](https://www.uvicorn.org/) (installed via requirements.txt)

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
