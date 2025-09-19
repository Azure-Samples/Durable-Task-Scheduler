# Human Interaction Pattern - Durable Functions with Durable Task Scheduler

This sample demonstrates the **Human Interaction** orchestration pattern using Durable Functions with the Durable Task Scheduler backend. This pattern shows how to handle workflows that require human approval or input with timeout handling.

## Pattern Overview

The Human Interaction pattern waits for human input with timeout handling:
1. **Process Start**: `send_approval_request` - Sends a request for human approval
2. **Wait for Input**: Orchestration waits for external event or timeout
3. **Timeout Handling**: If no response within timeout, escalates or takes default action
4. **Completion**: Processes the approval/denial or timeout result

## Architecture

- **HTTP Trigger**: `request_approval` - Starts the approval workflow
- **Orchestrator**: `human_interaction_orchestrator` - Manages waiting and timeout logic
- **Activities**:
  - `send_approval_request` - Initiates the approval request
  - `process_approval` - Handles approved requests  
  - `escalate_approval` - Handles timeout scenarios
- **Event Endpoint**: `approve/{instanceId}` - Receives approval/denial
- **Backend**: Durable Task Scheduler for state management

## Prerequisites

- [Python 3.9+](https://www.python.org/downloads/)
- [Azure Functions Core Tools](https://docs.microsoft.com/azure/azure-functions/functions-run-local#install-the-azure-functions-core-tools)
- [Durable Task Scheduler Emulator](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler) (for local development)

## Setup

1. **Install dependencies**:
   ```bash
   pip install -r requirements.txt
   ```

2. **Start the Durable Task Scheduler Emulator**:
   ```bash
   docker run --name dts-emulator -p 8080:8080 -p 8082:8082 -d mcr.microsoft.com/dts/dts-emulator:latest
   ```

3. **Configure connection** (already set in `local.settings.json`):
   The sample is configured to use the local emulator by default.

## Running the Sample

1. **Start the Azure Functions host**:
   ```bash
   func start
   ```

2. **Start an approval workflow**:
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

3. **Provide approval/denial** (within the timeout period):
   ```bash
   # Approve the request (use approvalId from the orchestration response)
   curl -X POST http://localhost:7071/api/approve/{approvalId} \
     -H "Content-Type: application/json" \
     -d '{"approved": true, "comments": "Approved by manager", "approver": "manager@company.com"}'

   # Deny the request  
   curl -X POST http://localhost:7071/api/approve/{approvalId} \
     -H "Content-Type: application/json" \
     -d '{"approved": false, "comments": "Insufficient justification", "approver": "manager@company.com"}'
   ```

4. **Check status**:
   ```bash
   curl http://localhost:7071/api/status/{instanceId}
   ```

## Configuration Files

### host.json
Configures the Durable Functions extension to use Durable Task Scheduler:
- Sets the hub name to "default"
- Configures the storage provider as "azureManaged"
- References the connection string name

### local.settings.json
Contains local development settings:
- Durable Task Scheduler connection string for local emulator
- Function worker runtime set to "python"

## Expected Behavior

### Scenario 1: Approved Within Timeout
```json
{
  "requestType": "ExpenseApproval", 
  "status": "Approved",
  "approval": {
    "approved": true,
    "comments": "Approved by manager",
    "timestamp": "2025-09-19T16:45:30Z"
  }
}
```

### Scenario 2: Denied Within Timeout
```json
{
  "requestType": "ExpenseApproval",
  "status": "Denied", 
  "approval": {
    "approved": false,
    "comments": "Insufficient justification",
    "timestamp": "2025-09-19T16:45:30Z"
  }
}
```

### Scenario 3: Timeout (No Response)
```json
{
  "requestType": "ExpenseApproval",
  "status": "Escalated",
  "reason": "No response within timeout period",
  "escalation": "Sent to senior management"
}
```

## How It Works

1. **Timer Setup**: Orchestration sets up a durable timer for the timeout period
2. **Event Waiting**: Waits for either external approval event or timer completion
3. **Race Condition**: Whichever completes first (approval or timeout) determines the outcome
4. **State Management**: Durable Task Scheduler maintains state during the waiting period

## Testing Timeout Scenarios

To test timeout behavior:
1. Start an approval request with a short timeout (e.g., 10 seconds)
2. Don't send any approval/denial
3. Wait for the timeout to occur
4. Check the final status to see escalation handling

## Monitoring

- **Function Logs**: Check the Azure Functions host output for approval workflow details
- **Dashboard**: Navigate to http://localhost:8082 to view waiting orchestrations
- **Timers**: See active timers and their remaining durations in the dashboard

## Using with Azure Durable Task Scheduler

To use with an Azure-hosted Durable Task Scheduler instead of the emulator:

1. Update `local.settings.json`:
   ```json
   {
     "Values": {
       "DURABLE_TASK_SCHEDULER_CONNECTION_STRING": "Endpoint=https://your-scheduler.dts.azure.net;Authentication=DefaultAzure"
     }
   }
   ```

2. Ensure you're authenticated with Azure CLI:
   ```bash
   az login
   ```

## Learn More

- [Durable Functions Overview](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-overview)
- [Durable Task Scheduler](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler)
- [Human Interaction Pattern](https://docs.microsoft.com/azure/azure-functions/durable/durable-functions-phone-verification)