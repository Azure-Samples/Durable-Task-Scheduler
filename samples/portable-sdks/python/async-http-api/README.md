# Async HTTP API Pattern

This sample demonstrates the async HTTP API pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern allows you to start long-running operations via HTTP and retrieve their results once they complete, without forcing clients to wait synchronously.

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
3. [Docker](https://www.docker.com/products/docker-desktop/) (for emulator option)

## Sample Overview

In this sample, the orchestration demonstrates the async HTTP API pattern by:

1. Starting a long-running operation asynchronously
2. Returning a status URL immediately to the client
3. Processing the request in the background
4. Making the result available for retrieval when complete

This pattern is ideal for implementing RESTful services with long-running operations, avoiding the need to keep HTTP connections open for extended periods.

## Configuring the Sample

There are two separate ways to run an example:

- Using the Emulator
- Using a deployed Scheduler and Taskhub

### Running with a Deployed Scheduler and Taskhub Resource

1. To create a taskhub, follow these steps using the Azure CLI commands:

Create a Scheduler:

```bash
az durabletask scheduler create --resource-group --name --location --ip-allowlist "[0.0.0.0/0]" --sku-capacity 1 --sku-name "Dedicated" --tags "{'myattribute':'myvalue'}"
```

Create Your Taskhub:

```bash
az durabletask taskhub create --resource-group <testrg> --scheduler-name <testscheduler> --name <testtaskhub>
```

2. Retrieve the Endpoint for the Scheduler: Locate the taskhub in the Azure portal to find the endpoint.

3. Set the Environment Variables:

Bash:
```bash
export TASKHUB=<taskhubname>
export ENDPOINT=<taskhubEndpoint>
```

Powershell:
```powershell
$env:TASKHUB = "<taskhubname>"
$env:ENDPOINT = "<taskhubEndpoint>"
```

4. Install the Correct Packages:
```bash
pip install -r requirements.txt
```

5. Grant your developer credentials the Durable Task Data Contributor Role.

### Running with the Emulator

The emulator simulates a scheduler and taskhub, packaged into an easy-to-use Docker container. For these steps, it is assumed that you are using port 8080.

1. Install Docker: If it is not already installed.

2. Pull the Docker Image for the Emulator:

```bash
docker pull mcr.microsoft.com/dts/dts-emulator:v0.0.6
```

3. Run the Emulator: Wait a few seconds for the container to be ready.

```bash
docker run --name dtsemulator -d -p 8080:8080 mcr.microsoft.com/dts/dts-emulator:v0.0.4
```

4. Set the Environment Variables:

Bash:
```bash
export TASKHUB=<taskhubname>
export ENDPOINT=http://localhost:8080
```

Powershell:
```powershell
$env:TASKHUB = "<taskhubname>"
$env:ENDPOINT = "http://localhost:8080"
```

5. Edit the Examples: Change the `token_credential` input of both the `DurableTaskSchedulerWorker` and `DurableTaskSchedulerClient` to `None`.

## Running the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. First, activate your Python virtual environment:
```bash
python -m venv venv
source venv/bin/activate  # On Windows, use: venv\Scripts\activate
```

2. Install the required packages:
```bash
pip install -r requirements.txt
```

3. Start the worker in a terminal:
```bash
python worker.py
```
You should see output indicating the worker has started and registered the orchestration and activities.

4. In a new terminal (with the virtual environment activated), run the FastAPI client:
```bash
python client.py
```
The FastAPI application will start on http://localhost:8000.

5. Interact with the API using curl or a web browser:

   - **Start a long-running operation:**
     ```
     curl -X POST http://localhost:8000/api/process \
          -H "Content-Type: application/json" \
          -d '{"name": "Your Name", "delay_seconds": 10}'
     ```
     This will return links for checking status and retrieving the result.

   - **Check operation status:**
     ```
     curl http://localhost:8000/api/status/{operation_id}
     ```
     Replace `{operation_id}` with the ID returned from the previous call.

   - **Get operation result (when completed):**
     ```
     curl http://localhost:8000/api/result/{operation_id}
     ```
     Replace `{operation_id}` with the appropriate operation ID.

### What Happens When You Run the Sample

When you run the sample:

1. The client creates a FastAPI web application that provides REST endpoints for starting operations and checking their status.

2. When you submit a processing request:
   - The client initiates a new orchestration instance
   - It immediately returns a response with status code 202 (Accepted)
   - The response includes URLs for checking status and retrieving results

3. The worker executes the `process_request` orchestration function, which:
   - Receives the processing request parameters
   - Calls the `simulate_long_running_activity` activity, which simulates work by sleeping
   - Completes and returns the final result

4. When you check the status, the API queries the current orchestration state and returns:
   - Whether the operation is running, completed, or failed
   - The current timestamp
   - Links to status and result endpoints

5. When you request the result (after completion), the API retrieves and returns the final output of the orchestration.

This sample demonstrates how to implement RESTful APIs for long-running operations using the Durable Task Scheduler, providing a better user experience by not requiring clients to maintain open connections while processing completes.

## Sample Explanation

The async HTTP API pattern is useful for implementing RESTful services with long-running operations. Instead of keeping an HTTP connection open for the entire operation, this pattern:

1. Returns an immediate response with a status URL
2. Processes the request asynchronously in the background
3. Allows the client to check the status via the provided URL
4. Returns the final result when the operation completes

This pattern is common in many real-world scenarios:
- File processing services
- Data import/export operations
- Complex calculations or analysis
- Resource provisioning

In this sample, the FastAPI application demonstrates how to use durable tasks to manage the lifecycle of asynchronous operations while providing a responsive HTTP API.
