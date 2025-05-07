# Fan-Out Fan-In Pattern

## Description of the Sample

This sample demonstrates the Fan-Out Fan-In pattern with the Azure Durable Task Scheduler using the .NET SDK. The Fan-Out Fan-In pattern represents a way to execute multiple operations in parallel and then aggregate the results, making it ideal for parallelized data processing scenarios.

In this sample:
1. The orchestrator takes a list of work items as input
2. It fans out by creating a separate task for each work item using `ProcessWorkItemActivity` 
3. All these tasks execute in parallel
4. It waits for all tasks to complete using `Task.WhenAll`
5. It fans in by aggregating all individual results using `AggregateResultsActivity`
6. The final aggregated result is returned to the client

This pattern is useful for:
- Processing large datasets in parallel for improved throughput
- Batch processing operations that can be executed independently
- Distributing computational workload across multiple workers
- Aggregating results from multiple sources or computations

## Prerequisites

1. [.NET 8 SDK](https://dotnet.microsoft.com/download/dotnet/8.0) or later
2. [Docker](https://www.docker.com/products/docker-desktop/) (for running the emulator)
3. [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) (if using a deployed Durable Task Scheduler)

## Configuring Durable Task Scheduler

There are two ways to run this sample:

### Using the Emulator (Recommended)

The emulator simulates a scheduler and taskhub in a Docker container, making it ideal for development and learning.

1. Install Docker if it's not already installed.

2. Pull the Docker Image for the Emulator:
```bash
docker pull mcr.microsoft.com/dts/dts-emulator:latest
```

3. Run the Emulator:
```bash
docker run --name dtsemulator -d -p 8080:8080 -p 8082:8082 mcr.microsoft.com/dts/dts-emulator:latest
```
Wait a few seconds for the container to be ready.

Note: The example code automatically uses the default emulator settings (endpoint: http://localhost:8080, taskhub: default). You don't need to set any environment variables.

### Using a Deployed Scheduler and Taskhub in Azure

For production scenarios or when you're ready to deploy to Azure:

1. Create a Scheduler using the Azure CLI:
```bash
az durabletask scheduler create --resource-group <testrg> --name <testscheduler> --location <eastus> --ip-allowlist "[0.0.0.0/0]" --sku-capacity 1 --sku-name "Dedicated" --tags "{'myattribute':'myvalue'}"
```

2. Create Your Taskhub:
```bash
az durabletask taskhub create --resource-group <testrg> --scheduler-name <testscheduler> --name <testtaskhub>
```

3. Retrieve the Endpoint for the Scheduler from the Azure portal.

4. Set the Environment Variables:

   Bash:
   ```bash
   export TASKHUB=<taskhubname>
   export ENDPOINT=<taskhubEndpoint>
   ```

   PowerShell:
   ```powershell
   $env:TASKHUB = "<taskhubname>"
   $env:ENDPOINT = "<taskhubEndpoint>"
   ```

## Authentication

The sample includes smart detection of the environment and configures authentication automatically:

- For local development with the emulator (when endpoint is http://localhost:8080), no authentication is required.
- For Azure deployments, DefaultAzure authentication is used, which utilizes DefaultAzureCredential behind the scenes and tries multiple authentication methods:
  - Managed Identity
  - Environment variables (AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_CLIENT_SECRET)
  - Azure CLI login
  - Visual Studio login
  - and more

The connection string is constructed dynamically based on the environment:
```csharp
// For local emulator
connectionString = $"Endpoint={hostAddress};TaskHub={taskHubName};Authentication=None";

// For Azure
connectionString = $"Endpoint={hostAddress};TaskHub={taskHubName};Authentication=DefaultAzureCredential";
```

## How to Run the Sample

Once you have set up either the emulator or deployed scheduler, follow these steps to run the sample:

1. First, build the solution:
```bash
cd FanOutFanIn
dotnet build
```

2. Start the worker in a terminal:
```bash
cd Worker
dotnet run
```
You should see output indicating the worker has started and registered the orchestration and activities.

3. In a new terminal, run the client:
```bash
cd Client
dotnet run
```

## Understanding the Code Structure

### Worker Project

The Worker project contains:

- **ParallelProcessingOrchestration.cs**: Defines the orchestrator and activity functions in a single file
- **Program.cs**: Sets up the worker host with proper connection string handling

#### Orchestration Implementation

The orchestration uses the fan-out fan-in pattern by creating parallel activity tasks and waiting for all to complete:

```csharp
public override async Task<Dictionary<string, int>> RunAsync(TaskOrchestrationContext context, List<string> workItems)
{
    // Step 1: Fan-out by creating a task for each work item in parallel
    List<Task<Dictionary<string, int>>> processingTasks = new List<Task<Dictionary<string, int>>>();
    
    foreach (string workItem in workItems)
    {
        // Create a task for each work item (fan-out)
        Task<Dictionary<string, int>> task = context.CallActivityAsync<Dictionary<string, int>>(
            nameof(ProcessWorkItemActivity), workItem);
        processingTasks.Add(task);
    }
    
    // Step 2: Wait for all parallel tasks to complete
    Dictionary<string, int>[] results = await Task.WhenAll(processingTasks);
    
    // Step 3: Fan-in by aggregating all results
    Dictionary<string, int> aggregatedResults = await context.CallActivityAsync<Dictionary<string, int>>(
        nameof(AggregateResultsActivity), results);
    
    return aggregatedResults;
}
```

Each activity is implemented as a separate class decorated with the `[DurableTask]` attribute:

```csharp
[DurableTask]
public class ProcessWorkItemActivity : TaskActivity<string, Dictionary<string, int>>
{
    // Implementation processes a single work item
}

[DurableTask]
public class AggregateResultsActivity : TaskActivity<Dictionary<string, int>[], Dictionary<string, int>>
{
    // Implementation aggregates individual results
}
```

The worker uses Microsoft.Extensions.Hosting for proper lifecycle management:
```csharp
builder.Services.AddDurableTaskWorker()
    .AddTasks(registry =>
    {
        registry.AddOrchestrator<ParallelProcessingOrchestration>();
        registry.AddActivity<ProcessWorkItemActivity>();
        registry.AddActivity<AggregateResultsActivity>();
    })
    .UseDurableTaskScheduler(connectionString);
```

### Client Project

The Client project:

- Uses the same connection string logic as the worker
- Creates a list of work items to be processed in parallel
- Schedules an orchestration instance with the list as input
- Waits for the orchestration to complete and displays the aggregated results
- Uses WaitForInstanceCompletionAsync for efficient polling

```csharp
List<string> workItems = new List<string>
{
    "Task1",
    "Task2",
    "Task3",
    "LongerTask4",
    "VeryLongTask5"
};

// Schedule the orchestration with the work items
string instanceId = await client.ScheduleNewOrchestrationInstanceAsync(
    "ParallelProcessingOrchestration", 
    workItems);

// Wait for completion
var instance = await client.WaitForInstanceCompletionAsync(
    instanceId,
    getInputsAndOutputs: true,
    cts.Token);
```

## Understanding the Output

When you run the client, you should see:
1. The client starting an orchestration with a list of work items
2. The worker processing each work item in parallel
3. The worker aggregating all results
4. The client displaying the final aggregated results from the completed orchestration

Example output:
```
Starting Fan-Out Fan-In Pattern - Parallel Processing Client
Using local emulator with no authentication
Starting parallel processing orchestration with 5 work items
Work items: ["Task1","Task2","Task3","LongerTask4","VeryLongTask5"]
Started orchestration with ID: 7f8e9a6b-1c2d-3e4f-5a6b-7c8d9e0f1a2b
Waiting for orchestration to complete...
Orchestration completed with status: Completed
Processing results:
Work item: Task1, Result: 5
Work item: Task2, Result: 5
Work item: Task3, Result: 5
Work item: LongerTask4, Result: 11
Work item: VeryLongTask5, Result: 13
Total items processed: 5
```

When you run the sample, you'll see output from both the worker and client processes:

### Worker Output
The worker shows:
- Registration of the orchestrator and activities
- Log entries when each activity is called
- Parallel processing of multiple work items
- Final aggregation of results

### Client Output
The client shows:
- Starting the orchestration with a list of work items
- The unique orchestration instance ID
- The final aggregated results, showing each work item and its corresponding result
- Total count of processed items

This demonstrates the power of the Fan-Out Fan-In pattern for parallel processing and result aggregation.

## Reviewing the Orchestration in the Durable Task Scheduler Dashboard

To access the Durable Task Scheduler Dashboard and review your orchestration:

### Using the Emulator
1. Navigate to http://localhost:8082 in your web browser
2. Click on the "default" task hub
3. You'll see the orchestration instance in the list
4. Click on the instance ID to view the execution details, which will show:
   - The parallel execution of multiple activity tasks
   - The fan-in aggregation step
   - The input and output at each step
   - The time taken for each step

### Using a Deployed Scheduler
1. Navigate to the Scheduler resource in the Azure portal
2. Go to the Task Hub subresource that you're using
3. Click on the dashboard URL in the top right corner
4. Search for your orchestration instance ID
5. Review the execution details

The dashboard visualizes the Fan-Out Fan-In pattern, making it easy to see how tasks are distributed in parallel and then aggregated back together.
