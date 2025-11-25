# Human Interaction Pattern

## Description of the Sample

This sample demonstrates the human interaction pattern with Azure Durable Functions in Python. This pattern is essential for workflows that require human approval, input, or decision-making with built-in timeout handling to prevent indefinite waiting.

In this sample:
1. **Approval Request**: The orchestrator starts by sending an approval request using the `send_approval_request` activity
2. **External Event Waiting**: The orchestration waits for an external event (approval/denial) or a timeout
3. **Race Condition**: Either the human provides input or the timeout expires - whichever happens first determines the flow
4. **Response Processing**: Approved requests are processed via `process_approval`, while timeouts trigger `escalate_approval`
5. **Final Result**: The orchestration completes with the approval status or escalation details

This pattern is useful for:
- Approval workflows for expenses, time-off requests, or purchases
- Document review and sign-off processes  
- Quality control checkpoints requiring human validation
- Any process where automated systems need human oversight with timeout handling

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

4. Start an approval workflow by sending a POST request:
   ```bash
   # Start approval request with default timeout (10 minutes)
   curl -X POST http://localhost:7071/api/human_interaction \
     -H "Content-Type: application/json" \
     -d '{}'

   # Start approval request with custom parameters
   curl -X POST http://localhost:7071/api/human_interaction \
     -H "Content-Type: application/json" \
     -d '{"request_type": "expense_approval", "amount": 5000, "timeout_minutes": 15, "requester": "alice@company.com", "description": "Conference attendance expenses"}'
   ```

5. Provide approval or denial within the timeout period:
   ```bash
   # Approve the request (use approvalId from the orchestration response)
   curl -X POST http://localhost:7071/api/approve/{approvalId} \
     -H "Content-Type: application/json" \
     -d '{"approved": true, "comments": "Approved by manager", "approver": "manager@company.com"}'

   # Or deny the request  
   curl -X POST http://localhost:7071/api/approve/{approvalId} \
     -H "Content-Type: application/json" \
     -d '{"approved": false, "comments": "Insufficient justification", "approver": "manager@company.com"}'
   ```

6. Check orchestration status:
   ```bash
   curl http://localhost:7071/api/status/{instanceId}
   ```

## Understanding the Output

When you run the sample, you'll see the following behavior based on different scenarios:

1. **Initial Response**: The HTTP trigger returns management URLs immediately:
   ```json
   {
     "id": "abcd1234",
     "approvalId": "approval-abcd1234",
     "statusQueryGetUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234",
     "sendEventPostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234/raiseEvent/{eventName}",
     "terminatePostUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234/terminate?reason={text}",
     "purgeHistoryDeleteUri": "http://localhost:7071/runtime/webhooks/durabletask/instances/abcd1234"
   }
   ```

2. **Scenario 1 - Approved Within Timeout**: If approval is provided before timeout expires:
   ```json
   {
     "requestType": "ExpenseApproval", 
     "status": "Approved",
     "approval": {
       "approved": true,
       "comments": "Approved by manager",
       "approver": "manager@company.com",
       "timestamp": "2025-09-19T16:45:30Z"
     }
   }
   ```

3. **Scenario 2 - Denied Within Timeout**: If denial is provided before timeout expires:
   ```json
   {
     "requestType": "ExpenseApproval",
     "status": "Denied", 
     "approval": {
       "approved": false,
       "comments": "Insufficient justification",
       "approver": "manager@company.com",
       "timestamp": "2025-09-19T16:45:30Z"
     }
   }
   ```

4. **Scenario 3 - Timeout (No Response)**: If no response is received within the timeout period:
   ```json
   {
     "requestType": "ExpenseApproval",
     "status": "Escalated",
     "reason": "No response within timeout period",
     "escalation": "Sent to senior management",
     "timeoutMinutes": 15
   }
   ```

5. **Testing Timeout Scenarios**: To test timeout behavior:
   - Start an approval request with a short timeout (e.g., 10 seconds)
   - Don't send any approval/denial response
   - Wait for the timeout to occur and check the final escalation status

## Dashboard Review

You can monitor the orchestration execution through the Durable Task Scheduler dashboard:

1. Navigate to `http://localhost:8082` in your browser
2. You'll see a list of task hubs - select the "default" hub
3. Click on your orchestration instance to see:
   - The orchestration waiting for external events (approval/denial)
   - Active durable timers and their remaining time until timeout
   - Timeline showing the approval request activity and the waiting state
   - Real-time updates when the external event is received or timeout occurs

The dashboard is particularly valuable for this pattern because it clearly visualizes the "waiting" state and shows how external events can interrupt long-running waits, demonstrating the human interaction pattern in real-time.

## Learn More

- [Human Interaction Pattern in Durable Functions](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-phone-verification)
- [Durable Task Scheduler Overview](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
- [Durable Functions Python Developer Guide](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-python)