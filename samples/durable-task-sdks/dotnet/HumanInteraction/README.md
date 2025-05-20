# Human Interaction Pattern

## Description of the Sample

This sample demonstrates the Human Interaction pattern with the Azure Durable Task Scheduler using the .NET SDK. This pattern is used for workflows that require human approval or input before continuing.

In this sample:
1. The orchestrator submits an approval request using the `SubmitApprovalRequestActivity`
2. It then waits for either an external event (approval response) or a timeout
3. When it receives a response or times out, it calls the `ProcessApprovalActivity`
4. The final result includes the approval status and related information

This pattern is useful for:
- Approval workflows (expense reports, document reviews, change requests)
- Business processes that require human decision making
- Multi-step processes with human validation steps
- Implementing timeouts for human response

## Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0) or later
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator) installed
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample locally: locally:

### Using the Emulator (Recommended)

The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. Pull the Docker Image for the Emulator:
    ```bash
    docker pull mcr.microsoft.com/dts/dts-emulator:latest
    ```

1. Run the Emulator:
    ```bash
    docker run -it -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
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

## Authentication

The sample includes smart detection of the environment and configures authentication automatically:

- For local development with the emulator (when endpoint is http://localhost:8080), no authentication is required.
- For local development with a deployed scheduler, DefaultAzure authentication is used, which utilizes DefaultAzureCredential behind the scenes and tries multiple authentication methods:
  - Managed Identity
  - Environment variables (AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_CLIENT_SECRET)
  - Azure CLI login
  - Visual Studio login
  - and more

The connection string is constructed dynamically based on the environment:
```csharp
// For local emulator
connectionString = $"Endpoint={schedulerEndpoint};TaskHub={taskHubName};Authentication=None";

// For Azure deployed emulator
connectionString = $"Endpoint={schedulerEndpoint};TaskHub={taskHubName};Authentication=DefaultAzure";
```

## How to Run the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1.  If you're using a deployed scheduler, you need to set Environment Variables.
    ```bash
    export ENDPOINT=$(az durabletask scheduler show \
        --resource-group my-resource-group \
        --name my-scheduler \
        --query "properties.endpoint" \
        --output tsv)

    export TASKHUB="my-taskhub"
    ```

1. Start the worker in a terminal:
    ```bash
    cd samples/durable-task-sdks/dotnet/HumanInteraction/Worker
    dotnet run
    ```
    You should see output indicating the worker has started and registered the orchestration and activities.

1. In a new terminal, run the client:
    > **Note:** Remember to set the environment variables again if you're using a deployed scheduler. 

    ```bash
    cd samples/durable-task-sdks/dotnet/HumanInteraction/Client
    dotnet run
    ```

    This will launch an interactive console client that creates an approval request and waits for your response.

## Identity-based authentication

Learn how to set up [identity-based authentication](https://learn.microsoft.com/azure/azure-functions/durable/durable-task-scheduler/durable-task-scheduler-identity?tabs=df&pivots=az-cli) when you deploy the app Azure.  

## Understanding the Code Structure

### Worker Project

The Worker project contains:

- **ApprovalOrchestration.cs**: Defines the orchestrator and activity functions in a single file
- **Program.cs**: Sets up the worker host with proper connection string handling

#### Orchestration Implementation

The orchestration implements the human interaction pattern by:

```csharp
public override async Task<ApprovalResult> RunAsync(TaskOrchestrationContext context, ApprovalRequestData input)
{
    // Submit the approval request
    SubmissionResult submissionResult = await context.CallActivityAsync<SubmissionResult>(
        nameof(SubmitApprovalRequestActivity), 
        requestData);
    
    // Make the status available via custom status
    context.SetCustomStatus(submissionResult);
    
    // Create a durable timer for the timeout
    Task timeoutTask = context.CreateTimer(timeoutDeadline, timeoutCts.Token);
    
    // Wait for an external event (approval/rejection)
    Task<ApprovalResponseData> approvalTask = context.WaitForExternalEvent<ApprovalResponseData>(approvalEventName);
    
    // Wait for either the timeout or the approval response, whichever comes first
    Task completedTask = await Task.WhenAny(approvalTask, timeoutTask);
    
    // Process based on which task completed
    if (completedTask == approvalTask)
    {
        // Human responded in time - process the approval
        ApprovalResponseData approvalData = approvalTask.Result;
        result = await context.CallActivityAsync<ApprovalResult>(
            nameof(ProcessApprovalActivity),
            /* approval details */);
    }
    else
    {
        // Timeout occurred
        result = /* timeout result */;
    }
    
    return result;
}
```

Activities are implemented as separate classes decorated with the `[DurableTask]` attribute:

```csharp
[DurableTask]
public class SubmitApprovalRequestActivity : TaskActivity<ApprovalRequestData, SubmissionResult>
{
    // Implementation for submitting the approval request
}

[DurableTask]
public class ProcessApprovalActivity : TaskActivity<dynamic, ApprovalResult>
{
    // Implementation for processing the approval result
}
```

The worker uses Microsoft.Extensions.Hosting for proper lifecycle management:
```csharp
builder.Services.AddDurableTaskWorker()
    .AddTasks(registry =>
    {
        registry.AddOrchestrator<ApprovalOrchestration>();
        registry.AddActivity<SubmitApprovalRequestActivity>();
        registry.AddActivity<ProcessApprovalActivity>();
    })
    .UseDurableTaskScheduler(connectionString);
```

### Client Project

The Client project:

- Uses the same connection string logic as the worker
- Creates a new approval request with a unique ID
- Schedules an orchestration instance with the request details
- Prompts the user to approve or reject the request
- Sends the response as an external event to the orchestration
- Checks the final status of the orchestration

```csharp
// Schedule the orchestration
string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
    "ApprovalOrchestration", 
    input,
    new OrchestrationInstanceOptions 
    { 
        InstanceId = requestId 
    });

// Wait for user decision
Console.WriteLine("\nPress Enter to approve the request, or type 'reject' and press Enter to reject: ");
string userInput = Console.ReadLine() ?? "";
bool isApproved = !userInput.Trim().Equals("reject", StringComparison.OrdinalIgnoreCase);

// Send approval response as an external event
await client.RaiseEventAsync(
    instanceId, 
    "approval_response", 
    approvalResponse);
```

## Understanding the Output

When you run the client, you should see:
1. The client starting an orchestration with approval request details
2. The worker processing the approval request
3. The client prompting you to approve or reject
4. The worker processing your response (or handling a timeout)
5. The client displaying the final result of the approval process

Example output:
```
Starting Human Interaction Pattern - Approval Client
Using local emulator with no authentication
Creating new approval request with ID: 7f8e9a6b-1c2d-3e4f-5a6b-7c8d9e0f1a2b
Request details: Requester: Console User, Item: Vacation Request, Timeout: 1h
Started orchestration with ID: 7f8e9a6b-1c2d-3e4f-5a6b-7c8d9e0f1a2b
Checking initial status...
  Status: Running
  Details:
  {
    "RequestId": "7f8e9a6b-1c2d-3e4f-5a6b-7c8d9e0f1a2b",
    "Status": "Pending",
    "SubmittedAt": "2023-04-16T14:32:00.0000000Z",
    "ApprovalUrl": "http://localhost:8000/api/approvals/7f8e9a6b-1c2d-3e4f-5a6b-7c8d9e0f1a2b"
  }

Press Enter to approve the request, or type 'reject' and press Enter to reject: 
Submitting your response (Approved)...
Waiting for final status...
Final status:
  Status: Completed
  Output:
  {
    "RequestId": "7f8e9a6b-1c2d-3e4f-5a6b-7c8d9e0f1a2b",
    "Status": "Approved",
    "ProcessedAt": "2023-04-16T14:32:15.0000000Z",
    "Approver": "Console User"
  }
Sample completed.
```

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
- Prompting you to approve or reject the request
- Submitting your response
- Final status showing the outcome (Approved, Rejected, or Timeout)

This demonstrates how a workflow can pause execution while waiting for human input, then continue processing once the input is received or a timeout occurs.

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestration:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see the orchestration instance in the list
4. Click on the instance ID to view the execution details, which will show:
   - The call to the `SubmitApprovalRequestActivity`
   - The waiting period for an external event
   - The potential parallel timeout task
   - The reception of the external event (if approved before timeout)
   - The call to the `ProcessApprovalActivity` with the decision
   - The final result

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details

The dashboard visualizes how the orchestration pauses while waiting for human input, showing the power of durable orchestrations to maintain state across long-running operations even when waiting for external events.
