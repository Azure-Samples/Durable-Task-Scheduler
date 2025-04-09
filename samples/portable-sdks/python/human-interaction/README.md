# Human Interaction Pattern

This sample demonstrates the human interaction pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern enables workflows that require manual approval or input from humans before proceeding.

## Prerequisites

1. [Python 3.8+](https://www.python.org/downloads/)
2. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli)
<<<<<<< HEAD
3. [Docker](https://www.docker.com/products/docker-desktop/) (for emulator option)

## Sample Overview

In this sample, the orchestration demonstrates the human interaction pattern by:

1. Submitting an approval request
2. Waiting for a human to approve or reject before continuing
3. Processing the approval response or handling a timeout if no response is received

This pattern is useful for scenarios where a workflow requires human input before proceeding.

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
=======
3. [Durable Task Scheduler resource](https://learn.microsoft.com/azure/durable-functions/durable-task-scheduler)
4. Appropriate Azure role assignments (Owner or Contributor)

## Setup

1. Create a virtual environment and activate it:

>>>>>>> 8b26beb (Add python samples for the durable app patterns)
```bash
python -m venv venv
source venv/bin/activate  # On Windows, use: venv\Scripts\activate
```

2. Install the required packages:
<<<<<<< HEAD
=======

>>>>>>> 8b26beb (Add python samples for the durable app patterns)
```bash
pip install -r requirements.txt
```

<<<<<<< HEAD
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
>>>>>>> 8b26beb (Add python samples for the durable app patterns)

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

<<<<<<< HEAD
### What Happens When You Run the Sample

When you run the sample:

1. The client creates a FastAPI web application that can start orchestrations and process approval responses.

2. When you create a new request, the client schedules a new orchestration instance with parameters including the requester, item, and timeout.

3. The worker executes the `approval_workflow` orchestration function, which:
   - Records the initial request details
   - Creates an external event name unique to this request
   - Waits for either an approval event or a timeout (using `wait_for_external_event` and `create_timer`)
   - Processes the approval decision or timeout and returns the final result

4. When you submit an approval or rejection via the API, the client raises an external event to the waiting orchestration with the approval details.

5. The orchestration processes the event and completes with the approval result.

This sample demonstrates how to incorporate human decision points into automated workflows, which is crucial for approval processes, review workflows, and other scenarios requiring human judgment.

## Viewing Orchestration Details in the Durable Task Dashboard

After running the sample, you can use the Durable Task Dashboard to view details about your human interaction orchestration:

1. Access the dashboard using the appropriate URL: `https://dashboard.durabletask.io`

2. When using an Azure deployed scheduler, you'll need to authenticate with an account that has been granted the "Durable Task Data Contributor" role.

3. In the dashboard, you'll see a list of all orchestration instances. Find your human interaction orchestration and click on it to see details.

4. The dashboard provides several views that are particularly useful for human interaction patterns:
   - **Timeline View**: Shows when the approval request was sent and when the human response was received
   - **History View**: Details the waiting period and response events
   - **Sequence View**: Visualizes the workflow steps including the human interaction point

The dashboard is particularly valuable for scenarios involving human interaction as it helps identify bottlenecks in approval processes and monitor overall workflow efficiency.
=======
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
>>>>>>> 8b26beb (Add python samples for the durable app patterns)
