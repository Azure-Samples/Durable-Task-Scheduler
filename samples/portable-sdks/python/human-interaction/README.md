# Human Interaction Pattern

This sample demonstrates the human interaction pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern enables workflows that require manual approval or input from humans before proceeding.

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

   - **Create an approval request:**
     ```
     curl -X POST http://localhost:8000/api/requests \
          -H "Content-Type: application/json" \
          -d '{"requester": "Alice", "item": "Vacation Request", "timeout_hours": 1}'
     ```
     This will return a request ID and approval URL.

   - **Check request status:**
     ```
     curl http://localhost:8000/api/requests/{request_id}
     ```
     Replace `{request_id}` with the ID returned from the previous call.

   - **Approve or reject the request:**
     ```
     curl -X POST http://localhost:8000/api/approvals/{request_id} \
          -H "Content-Type: application/json" \
          -d '{"is_approved": true, "approver": "Manager", "comments": "Approved!"}'
     ```
     Replace `{request_id}` with the appropriate request ID and set `is_approved` to `true` or `false`.

## Sample Explanation

The human interaction pattern is essential for workflows that require human approval or input before proceeding. Key aspects of this pattern include:

1. Submitting a request for human review
2. Suspending execution while waiting for a response
3. Handling responses (approval/rejection) when received
4. Managing timeouts if no response is received within a designated period

Common use cases include:
- Expense approval workflows
- Content moderation systems
- Change management processes
- Access request approvals

In this sample, the orchestration submits an approval request and then waits for either a human response (approve/reject) or a timeout. The FastAPI application provides endpoints for creating requests and responding to them, simulating a real-world approval system.
