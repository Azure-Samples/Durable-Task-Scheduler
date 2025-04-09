# Async HTTP API Pattern

This sample demonstrates the async HTTP API pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern allows you to start long-running operations via HTTP and retrieve their results once they complete, without forcing clients to wait synchronously.

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
3. [Durable Task Scheduler resource](https://learn.microsoft.com/azure/durable-functions/durable-task-scheduler)
4. Appropriate Azure role assignments (Owner or Contributor)

## Setup

1. Create a virtual environment and activate it:

```bash
python -m venv venv
source venv/bin/activate  # On Windows, use: venv\Scripts\activate
```

2. Install the required packages:

```bash
pip install -r requirements.txt
```

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
