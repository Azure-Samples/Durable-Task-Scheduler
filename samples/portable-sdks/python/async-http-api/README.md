# Async HTTP API Pattern

This sample demonstrates the async HTTP API pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern allows you to start long-running operations via HTTP and retrieve their results once they complete, without forcing clients to wait synchronously.

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
<<<<<<< HEAD
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
<<<<<<< HEAD
=======
3. [Durable Task Scheduler resource](https://learn.microsoft.com/azure/durable-functions/durable-task-scheduler)
4. Appropriate Azure role assignments (Owner or Contributor)

## Setup

1. Create a virtual environment and activate it:

>>>>>>> 8b26beb (Add python samples for the durable app patterns)
=======
>>>>>>> e8c58d1 (Continue improving READMEs)
```bash
python -m venv venv
source venv/bin/activate  # On Windows, use: venv\Scripts\activate
```

2. Install the required packages:
<<<<<<< HEAD
<<<<<<< HEAD
=======

>>>>>>> 8b26beb (Add python samples for the durable app patterns)
=======
>>>>>>> e8c58d1 (Continue improving READMEs)
```bash
pip install -r requirements.txt
```

<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> e8c58d1 (Continue improving READMEs)
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

<<<<<<< HEAD
### Viewing Orchestration Details in the Durable Task Dashboard

After running the sample, you can use the Durable Task Dashboard to view details about your orchestration execution:

1. Access the dashboard using the appropriate URL: `https://dashboard.durabletask.io`

2. When using an Azure deployed scheduler, you'll need to authenticate with an account that has been granted the "Durable Task Data Contributor" role.

3. In the dashboard, you'll see a list of all orchestration instances. Find your async HTTP API orchestration and click on it to see details.

4. The dashboard provides several views of your orchestration:
   - **Timeline View**: Shows the execution timeline with HTTP-triggered activities
   - **History View**: Provides detailed event sequence with timestamps
   - **Sequence View**: Visualizes the event sequence in a graphical format
   - **Input/Output View**: Shows the inputs to your orchestration and the final output

5. You can also see the status (Running, Completed, Failed) and manage orchestrations using the dashboard controls.

The dashboard is particularly useful for async HTTP patterns as it provides visibility into the progression of orchestrations triggered by HTTP requests and helps diagnose any issues with external API calls.

=======
3. Make sure you're logged in to Azure:

```bash
az login
```

4. Set up the required environment variables:

```bash
# For bash/zsh
export TASKHUB="your-taskhub-name"
export ENDPOINT="your-scheduler-endpoint"

# For Windows PowerShell
$env:TASKHUB="your-taskhub-name"
$env:ENDPOINT="your-scheduler-endpoint"
```

## Running the Sample

1. First, start the worker that registers the activities and orchestrations:

```bash
python worker.py
```

2. In a new terminal (with the virtual environment activated), run the FastAPI client:

```bash
python client.py
```

3. The FastAPI application will start on http://localhost:8000. You can interact with it using:

   - **Start an operation:**
     ```
     curl -X POST http://localhost:8000/api/start-operation \
          -H "Content-Type: application/json" \
          -d '{"processing_time": 10}'
     ```
     This will return an operation ID and status URL.

   - **Check operation status:**
     ```
     curl http://localhost:8000/api/operations/{operation_id}
     ```
     Replace `{operation_id}` with the ID returned from the previous call.

=======
>>>>>>> e8c58d1 (Continue improving READMEs)
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
>>>>>>> 8b26beb (Add python samples for the durable app patterns)
