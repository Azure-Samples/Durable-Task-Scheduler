# Human Interaction Pattern

## Description of the Sample

This sample demonstrates the human interaction pattern with the Azure Durable Task Scheduler using the Python SDK. This pattern is used for workflows that require human approval or input before continuing.

In this sample:
1. The orchestrator submits an approval request using the `submit_approval_request` activity
2. It then waits for either an external event (approval response) or a timeout
3. When it receives a response or times out, it calls the `process_approval` activity
4. The final result includes the approval status and related information

This pattern is useful for:
- Approval workflows (expense reports, document reviews, change requests)
- Business processes that require human decision making
- Multi-step processes with human validation steps
- Implementing timeouts for human response

## Prerequisites

1. [Python 3.9+](https://www.python.org/downloads/)
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator) installed
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

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

1. In a new terminal (with the virtual environment activated if applicable), run the client:
  > **Note:** Remember to set the environment variables again if you're using a deployed scheduler. 

   ```bash
   python client.py
   ```
   This will launch an interactive console client that creates an approval request and waits for your response.

## Understanding the Output

When you run the sample, you'll see output from both the worker and client processes:

### Worker Output
The worker shows:
- Registration of the orchestrator and activities
- Logging when the approval request is submitted
- The orchestrator waiting for an external event (your approval)
- Processing of the approval once received (or timeout handling)

### Client Output
The client shows:
- Creating a new approval request with a unique ID
- Initial status of the request (Pending)
- Prompting you to approve or reject the request (press Enter to approve, type "reject" to reject)
- Submitting your response
- Final status showing the outcome (Approved, Rejected, or Timeout)

The example demonstrates how a workflow can pause execution while waiting for human input, then continue processing once the input is received or a timeout occurs.

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestration:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see the orchestration instance in the list
4. Click on the instance ID to view the execution details, which will show:
   - The call to the `submit_approval_request` activity
   - The waiting period for an external event
   - The potential parallel timeout task
   - The reception of the external event (if approved before timeout)
   - The call to the `process_approval` activity with the decision
   - The final result

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details

The dashboard visualizes how the orchestration pauses while waiting for human input, showing the power of durable orchestrations to maintain state across long-running operations even when waiting for external events.
